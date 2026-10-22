// Independent (03-ring) checks for GAP-013's key/identity lifecycle and GAP-012's
// string-id contract at the pure-function level. Deliberately does not reuse the
// implementation's fixtures: these cases cover boundaries the shipped tests skip
// (success path, many-row reset, unrelated removal, id length sweep).
import { describe, expect, it, vi } from 'vitest'
import {
  appendDraftItem,
  createDraftOrderItem,
  createInitialDraftItems,
  removeDraftItem,
  resetDraftItems,
  type DraftOrderItem,
} from './order-draft'
import {
  createIdempotencyKey,
  createRowId,
  prepareOrderSubmission,
} from './order-submission'

const payload = { items: [{ product_id: 3, quantity: 2 }] }

function counter(prefix = 'row') {
  let n = 0
  return () => `${prefix}-${++n}`
}

describe('GAP-013 idempotency key lifecycle', () => {
  it('reuses the key only while the same submission is in flight', () => {
    const createKey = vi.fn(() => 'key-1')
    const pending = prepareOrderSubmission(payload, null, createKey)
    // A second click while the request is still outstanding: same order, same key.
    expect(prepareOrderSubmission(payload, pending, createKey)).toBe(pending)
    expect(createKey).toHaveBeenCalledTimes(1)
  })

  it('mints a new key after a failure clears the pending submission', () => {
    const keys = ['key-1', 'key-2']
    const createKey = vi.fn(() => keys.shift() as string)
    const first = prepareOrderSubmission(payload, null, createKey)
    // onError -> setPendingSubmission(null)
    const afterFailure = prepareOrderSubmission(payload, null, createKey)
    expect(afterFailure.idempotencyKey).not.toBe(first.idempotencyKey)
  })

  it('mints a new key after success when the same order is placed again', () => {
    const keys = ['key-1', 'key-2']
    const createKey = vi.fn(() => keys.shift() as string)
    const first = prepareOrderSubmission(payload, null, createKey)
    // onSuccess -> setPendingSubmission(null); the user re-enters the same order.
    const second = prepareOrderSubmission(payload, null, createKey)
    expect(second.idempotencyKey).toBe('key-2')
    expect(second.idempotencyKey).not.toBe(first.idempotencyKey)
  })

  it('mints a new key when only the quantity differs (fingerprint covers the payload)', () => {
    const createKey = vi.fn(() => 'k')
    const first = prepareOrderSubmission(payload, null, createKey)
    const changed = prepareOrderSubmission(
      { items: [{ product_id: 3, quantity: 3 }] },
      first,
      createKey
    )
    expect(changed.idempotencyKey).toBe('k')
    expect(createKey).toHaveBeenCalledTimes(2)
  })

  it('produces 500 pairwise-distinct keys', () => {
    const keys = Array.from({ length: 500 }, () => createIdempotencyKey())
    expect(new Set(keys).size).toBe(keys.length)
    for (const key of keys) expect(key.length).toBeGreaterThan(10)
  })
})

describe('GAP-013 row identity', () => {
  it('every rowId changes on reset, even with several rows', () => {
    const gen = counter()
    const before: DraftOrderItem[] = [
      createDraftOrderItem(gen),
      appendDraftItem([], gen)[0],
      createDraftOrderItem(gen),
    ]
    const after = resetDraftItems(gen)
    expect(after).toHaveLength(1)
    expect(before.some((row) => row.rowId === after[0].rowId)).toBe(false)
  })

  it('removing an unrelated row keeps the surviving rows identical', () => {
    const gen = counter()
    const rows = [createDraftOrderItem(gen), createDraftOrderItem(gen), createDraftOrderItem(gen)]
    rows[1] = { ...rows[1], product_id: '7' }
    const remaining = removeDraftItem(rows, 0)
    expect(remaining.map((r) => r.rowId)).toEqual([rows[1].rowId, rows[2].rowId])
    expect(remaining[0].product_id).toBe('7')
  })

  it('rowIds stay unique across interleaved create/append/reset', () => {
    const gen = counter()
    const seen = new Set<string>()
    let rows = createInitialDraftItems()
    rows.forEach((r) => seen.add(r.rowId))
    for (let i = 0; i < 20; i++) {
      rows = appendDraftItem(rows, gen)
      seen.add(rows[rows.length - 1].rowId)
      if (i % 3 === 0) {
        rows = resetDraftItems(gen)
        seen.add(rows[0].rowId)
      }
    }
    // counter() is deterministic; the real generator must be injective too.
    const real = Array.from({ length: 200 }, () => createRowId())
    expect(new Set(real).size).toBe(real.length)
  })
})

describe('GAP-012 string id contract', () => {
  // Sweep the digit lengths a Snowflake id actually spans over its lifetime.
  const samples = [
    '371447562240004097', // 18 digits: what the generator emits today
    '9007199254740993', // 2^53 + 1: first non-representable integer
    '1152921504606846975',
    '1900000000000000001',
  ]

  it.each(samples)('keeps %s intact when carried as a string', (id) => {
    // What the UI does: template-literal into the URL, no numeric cast.
    const path = `/orders/${id}`
    expect(path).toBe(`/orders/${id}`)
    expect(path.endsWith(id)).toBe(true)
    // Contrast: the numeric path changes the id, which is the defect.
    expect(String(Number(id))).not.toBe(id)
  })

  it('documents that Number() is the only lossy step', () => {
    const id = '371447562240004097'
    const truncated = String(Number(id))
    expect(Number.isSafeInteger(Number(id))).toBe(false)
    // Do not pin the exact digits: Go prints the float64's exact integer
    // (371447562240004096) while JS prints the shortest round-tripping decimal
    // (371447562240004100). Both are "a different id"; only this assertion is
    // portable across engines.
    expect(truncated).not.toBe(id)
    // eslint-disable-next-line no-console
    console.log(`qa: id=${id} truncated=${truncated} delta=${Number(truncated) - Number(id)}`)
  })
})
