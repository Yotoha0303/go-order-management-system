// Independent (03-ring) component-level checks for GAP-013. The shipped
// orders.test.tsx covers reset-after-select and one failure/retry path; these
// cover surfaces it leaves open: the Radix controlled/uncontrolled warning is
// checked on *every* transition (not only after reset), the reset entry point
// itself, and the "clear must not leave a stale idempotency key" path.
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { orderApi } from './api'
import { OrdersPage } from './orders'
import ordersSource from './orders.tsx?raw'

const products = [
  { id: 1, name: '测试商品A', description: '', price_fen: 1000, status: 1, created_at: '', updated_at: '' },
  { id: 2, name: '测试商品B', description: '', price_fen: 2000, status: 1, created_at: '', updated_at: '' },
]

vi.mock('./api', () => ({
  getErrorMessage: (error: unknown) => String(error),
  queryKeys: {
    ordersRoot: ['orders'],
    orders: (page: number, pageSize: number) => ['orders', { page, pageSize }],
    order: (id: string) => ['orders', id],
    products: (status: unknown) => ['products', { status }],
    stockLogsRoot: ['stock-logs'],
  },
  orderApi: {
    list: vi.fn(async () => ({ orders: [], total: 0, page: 1, page_size: 10 })),
    detail: vi.fn(async () => ({
      order: {
        id: '371447562240004097',
        order_no: 'QA-1',
        total_amount_fen: 1000,
        status: 1,
        created_at: '',
        updated_at: '',
      },
      items: [],
    })),
    pay: vi.fn(),
    finish: vi.fn(),
    cancel: vi.fn(),
    create: vi.fn(),
  },
  productApi: {
    list: vi.fn(async () => ({ products, total: products.length, page: 1, page_size: 100 })),
  },
}))

vi.mock('./components', () => ({
  BusinessPage: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Field: ({ label, children }: { label: string; children: React.ReactNode }) => (
    <div>
      <span>{label}</span>
      {children}
    </div>
  ),
  ApiErrorPanel: () => null,
  EmptyRow: () => null,
  LoadingRow: () => null,
  OrderStatusBadge: () => null,
}))

async function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return await render(
    <QueryClientProvider client={client}>
      <OrdersPage />
    </QueryClientProvider>
  )
}

function controlledWarnings(spy: { mock: { calls: unknown[][] } }) {
  return spy.mock.calls
    .map((call) => String(call[0]))
    .filter((message) => message.includes('uncontrolled to controlled'))
}

describe('OrdersPage GAP-013 independent checks', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(orderApi.create).mockReset()
  })

  // Why a source assertion: this package's Radix Select version does not warn
  // for `value={''}` -> string (verified by mutation), so the behavioural tests
  // in this file and in the implementation's orders.test.tsx both stay green if
  // the row key regresses to the array index. The remount is the fix here, so
  // the key expression itself must be pinned. This is the only check in this
  // file that fails on the index-key regression.
  it('keys draft rows by rowId, never by the array index', () => {
    expect(ordersSource).toContain('key={item.rowId}')
    expect(ordersSource).not.toMatch(/key=\{index\}/)
  })

  it('never flips the Select controlled/uncontrolled across select -> clear -> select', async () => {
    const warn = vi.spyOn(console, 'warn')
    try {
      const screen = await renderPage()
      const trigger = screen.getByRole('combobox').first()

      // Only the *first* interaction can reveal an uncontrolled start; if the
      // component began uncontrolled, this is where Radix warns.
      await userEvent.click(trigger)
      await userEvent.click(screen.getByRole('option', { name: /测试商品A/ }))
      await expect.element(trigger).toHaveTextContent(/测试商品A/)

      await userEvent.click(screen.getByRole('button', { name: '清空' }))
      await expect.element(trigger).toHaveTextContent(/请选择商品/)

      await userEvent.click(trigger)
      await userEvent.click(screen.getByRole('option', { name: /测试商品B/ }))
      await expect.element(trigger).toHaveTextContent(/测试商品B/)

      expect(controlledWarnings(warn)).toEqual([])
    } finally {
      warn.mockRestore()
    }
  })

  it('the 清空 button is a real reset entry point (clears the product and the quantity)', async () => {
    const screen = await renderPage()
    const trigger = screen.getByRole('combobox').first()

    await userEvent.click(trigger)
    await userEvent.click(screen.getByRole('option', { name: /测试商品A/ }))
    const quantity = screen.getByPlaceholder('1')
    await userEvent.fill(quantity, '9')

    await userEvent.click(screen.getByRole('button', { name: '清空' }))

    await expect.element(trigger).toHaveTextContent(/请选择商品/)
    await expect.element(quantity).toHaveValue('1')
    expect(screen.getByRole('button', { name: '清空' })).toBeTruthy()
  })

  it('retrying the same order after a failure mints a new idempotency key', async () => {
    const create = vi.mocked(orderApi.create)
    create.mockRejectedValueOnce(new Error('库存不足')).mockRejectedValueOnce(new Error('库存不足'))

    const screen = await renderPage()
    await userEvent.click(screen.getByRole('combobox').first())
    await userEvent.click(screen.getByRole('option', { name: /测试商品A/ }))

    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))
    await expect.poll(() => create.mock.calls.length).toBe(1)
    // Straight retry, without touching the form.
    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))
    await expect.poll(() => create.mock.calls.length).toBe(2)

    const [firstPayload, firstKey] = create.mock.calls[0]
    const [secondPayload, secondKey] = create.mock.calls[1]
    expect(secondPayload).toEqual(firstPayload)
    expect(secondKey).not.toBe(firstKey)
  })

  it('clearing after a failure does not resurrect the failed idempotency key', async () => {
    const create = vi.mocked(orderApi.create)
    create.mockRejectedValueOnce(new Error('库存不足')).mockResolvedValueOnce({
      id: '371447562240004097',
      order_no: 'QA-1',
      total_amount_fen: 1000,
      status: 1,
      created_at: '',
      updated_at: '',
    } as never)

    const screen = await renderPage()
    await userEvent.click(screen.getByRole('combobox').first())
    await userEvent.click(screen.getByRole('option', { name: /测试商品A/ }))

    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))
    await expect.poll(() => create.mock.calls.length).toBe(1)

    // User clears instead of retrying, then builds the same order again.
    await userEvent.click(screen.getByRole('button', { name: '清空' }))
    await userEvent.click(screen.getByRole('combobox').first())
    await userEvent.click(screen.getByRole('option', { name: /测试商品A/ }))
    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))
    await expect.poll(() => create.mock.calls.length).toBe(2)

    const [, firstKey] = create.mock.calls[0]
    const [, secondKey] = create.mock.calls[1]
    expect(secondKey).not.toBe(firstKey)
  })
})
