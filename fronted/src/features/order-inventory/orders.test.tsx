import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { orderApi } from './api'
import { OrdersPage } from './orders'
import type { Order, OrderDetail } from './types'

const product = {
  id: 1,
  name: '测试商品',
  description: '',
  price_fen: 1000,
  status: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

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
    detail: vi.fn(),
    pay: vi.fn(),
    finish: vi.fn(),
    cancel: vi.fn(),
    create: vi.fn(),
  },
  productApi: {
    list: vi.fn(async () => ({
      products: [product],
      total: 1,
      page: 1,
      page_size: 100,
    })),
  },
}))

// Minimal stand-ins: this test is about the create-order form, not the chrome.
vi.mock('./components', () => ({
  BusinessPage: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  Field: ({
    label,
    children,
  }: {
    label: string
    children: React.ReactNode
  }) => (
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

async function renderOrdersPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return await render(
    <QueryClientProvider client={client}>
      <OrdersPage />
    </QueryClientProvider>
  )
}

/** Radix emits this exact warning when `value` flips between undefined and a
 *  string. The fix (GAP-013) makes the Select controlled for its whole life, so
 *  it must never appear; catching it here turns the requirement into a
 *  falsifiable assertion instead of a line someone has to eyeball. */
function controlledSwitchWarnings(spy: {
  mock: { calls: unknown[][] }
}): string[] {
  return spy.mock.calls
    .map((call) => String(call[0]))
    .filter((message) => message.includes('uncontrolled to controlled'))
}

describe('OrdersPage create-order form (GAP-013)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Keep tests order-independent: only the retry test configures a rejection.
    vi.mocked(orderApi.create).mockReset()
  })

  // GAP-013 regression: the row used to be keyed by array index, so resetting
  // the draft kept key 0 alive and Radix Select kept displaying the previously
  // chosen product. With a per-row rowId the subtree remounts and the trigger
  // falls back to its placeholder.
  //
  // It also proves the Select stays controlled end to end: selecting, resetting
  // and selecting again must not emit Radix's uncontrolled->controlled warning.
  it('clears the selected product after reset and still accepts a new choice', async () => {
    const warn = vi.spyOn(console, 'warn')
    const screen = await renderOrdersPage()

    const trigger = screen.getByRole('combobox').first()
    await userEvent.click(trigger)
    await userEvent.click(screen.getByRole('option', { name: /测试商品/ }))

    await expect.element(trigger).toHaveTextContent(/测试商品/)

    await userEvent.click(screen.getByRole('button', { name: '清空' }))

    await expect.element(trigger).toHaveTextContent(/请选择商品/)
    await expect.element(trigger).not.toHaveTextContent(/测试商品/)

    // The reset must not break normal interaction: choosing the same product
    // again has to work, which it cannot if the Select is stuck uncontrolled.
    await userEvent.click(trigger)
    await userEvent.click(screen.getByRole('option', { name: /测试商品/ }))
    await expect.element(trigger).toHaveTextContent(/测试商品/)

    expect(controlledSwitchWarnings(warn)).toEqual([])
    warn.mockRestore()
  })

  // GAP-013 regression: onError used to leave pendingSubmission untouched, so
  // retrying the identical order reused the failed idempotency key and the
  // backend answered with an idempotent replay instead of creating the order.
  it('uses a fresh idempotency key after a failed submission', async () => {
    const create = vi.mocked(orderApi.create)
    create.mockRejectedValue(new Error('库存不足，创建失败'))

    const screen = await renderOrdersPage()

    const trigger = screen.getByRole('combobox').first()
    await userEvent.click(trigger)
    await userEvent.click(screen.getByRole('option', { name: /测试商品/ }))

    const submit = screen.getByRole('button', { name: '提交订单' })
    await userEvent.click(submit)
    await expect.poll(() => create.mock.calls.length).toBe(1)

    // Settings, inventory, logs and the transient error toast are irrelevant
    // here; the submit button is the stable handle for the second attempt.
    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))
    await expect.poll(() => create.mock.calls.length).toBe(2)

    const [firstPayload, firstKey] = create.mock.calls[0]
    const [secondPayload, secondKey] = create.mock.calls[1]

    expect(secondPayload).toEqual(firstPayload)
    expect(secondKey).not.toBe(firstKey)
  })

  // GAP-012 regression, and the reason this test has to exist at all: every
  // other case in this file lets create() reject (or leaves it un-stubbed), so
  // mutation.onSuccess never ran and nothing here observed what the component
  // does with the created order. The defect GAP-012 describes is a one-line
  // change *inside* onSuccess -- setSelectedOrderId(String(Number(order.id)))
  // -- and that change is invisible to every assertion above, because above
  // there is no successful creation to observe. So: force a real success with
  // a 19-digit Snowflake id and check the string that actually reaches the
  // detail API. The id is never coerced to number anywhere in the component;
  // Number() would truncate it (see the companion assertion) and the detail
  // request would ask for an order that does not exist.
  it('passes the created order id through to the detail query verbatim', async () => {
    // 19 digits, beyond Number.MAX_SAFE_INTEGER (2^53 - 1 = 16 digits of headroom).
    const createdId = '1058897343283204097'

    const createdOrder: Order = {
      id: createdId,
      order_no: 'ORD-20261022-0001',
      total_amount_fen: 1000,
      status: 1,
      created_at: '2026-10-22T00:00:00Z',
      updated_at: '2026-10-22T00:00:00Z',
    }

    // The detail loader is the observation point: its argument is the exact
    // string the component hands to orderApi.detail. It is stubbed through
    // mockImplementation (not mockResolvedValue) because the mock factory left
    // detail untyped, and this is also what types the call tuple below.
    const create = vi.mocked(orderApi.create)
    create.mockResolvedValue(createdOrder)
    vi.mocked(orderApi.detail).mockImplementation(async (id: string) => {
      void id
      return { order: createdOrder, items: [] } satisfies OrderDetail
    })

    const screen = await renderOrdersPage()

    await userEvent.click(screen.getByRole('combobox').first())
    await userEvent.click(screen.getByRole('option', { name: /测试商品/ }))
    await userEvent.click(screen.getByRole('button', { name: '提交订单' }))

    const detail = vi.mocked(orderApi.detail)
    await expect.poll(() => create.mock.calls.length).toBe(1)
    // The detail query only fires when selectedOrderId became non-null, i.e.
    // when onSuccess actually ran.
    await expect.poll(() => detail.mock.calls.length).toBe(1)

    // Strict identity on the raw string: not toEqual, not a length check, not
    // a substring match -- any of those would pass on the truncated id too.
    expect(detail.mock.calls[0][0]).toBe(createdId)
    // The assertion above only means something if this id *can* be damaged by
    // the coercion GAP-012 is about, so pin that premise in the same test; it
    // can never quietly stop being true. The truncated id is a different
    // string, which is exactly why the detail request used to 404.
    expect(String(Number(createdId))).not.toBe(createdId)
    expect(String(Number(createdId))).toBe('1058897343283204100')
  })

  // GAP-013-a behavioural pin for the row key. The tests above cannot catch a
  // regression to key={index}: since the Select became permanently controlled
  // (GAP-013 §4.6), Radix no longer warns, so the only thing left that index
  // keys break is *identity* -- the row subtree is reused instead of remounted.
  // So assert exactly that: the DOM node behind the first row's combobox after
  // a reset must not be the node that was there before it. Deliberately says
  // nothing about rowId naming, Radix warnings, or matcher behaviour -- the
  // identity is observed through the stable Locator.element() API.
  //
  // Note on the reference: once the reset happens the old node is detached and
  // any further query must go through a fresh locator. Only the captured
  // element reference is compared.
  it('replaces the draft row subtree with a new DOM node on reset', async () => {
    const screen = await renderOrdersPage()

    const triggerBefore = screen.getByRole('combobox').first()
    await userEvent.click(triggerBefore)
    await userEvent.click(screen.getByRole('option', { name: /测试商品/ }))
    await expect.element(triggerBefore).toHaveTextContent(/测试商品/)

    const nodeBeforeReset = triggerBefore.element()

    await userEvent.click(screen.getByRole('button', { name: '清空' }))

    // Re-query, do not reuse the stale locator: the old node may be detached.
    const triggerAfter = screen.getByRole('combobox').first()
    await expect.element(triggerAfter).toHaveTextContent(/请选择商品/)

    expect(triggerAfter.element()).not.toBe(nodeBeforeReset)
  })
})
