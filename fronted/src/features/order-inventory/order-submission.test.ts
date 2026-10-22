import { describe, expect, it, vi } from 'vitest'
import {
  createIdempotencyKey,
  createRowId,
  prepareOrderSubmission,
  type PendingOrderSubmission,
} from './order-submission'

const payload = { items: [{ product_id: 1, quantity: 2 }] }

describe('createIdempotencyKey', () => {
  it('falls back when randomUUID is unavailable (plain HTTP hosts)', () => {
    const original = globalThis.crypto
    Object.defineProperty(globalThis, 'crypto', {
      configurable: true,
      value: {
        getRandomValues: (bytes: Uint8Array) => {
          bytes.fill(7)
          return bytes
        },
      },
    })

    try {
      const key = createIdempotencyKey()
      expect(key.startsWith('idem-')).toBe(true)
      expect(key.length).toBeGreaterThan(10)
    } finally {
      Object.defineProperty(globalThis, 'crypto', {
        configurable: true,
        value: original,
      })
    }
  })
})

describe('createRowId', () => {
  it('never repeats, even within the same millisecond', () => {
    const ids = Array.from({ length: 50 }, () => createRowId())

    expect(new Set(ids).size).toBe(ids.length)
  })

  it('does not derive from the row index (0 and 1 stay distinct identities)', () => {
    expect(createRowId()).not.toBe(createRowId())
  })
})

describe('prepareOrderSubmission', () => {
  it('reuses the key when an unchanged request is retried while in flight', () => {
    const createKey = vi.fn(() => 'first-key')
    const first = prepareOrderSubmission(payload, null, createKey)
    const retry = prepareOrderSubmission(payload, first, createKey)

    expect(retry).toBe(first)
    expect(createKey).toHaveBeenCalledOnce()
  })

  it('creates a new key when the order content changes', () => {
    const createKey = vi
      .fn<() => string>()
      .mockReturnValueOnce('first-key')
      .mockReturnValueOnce('second-key')
    const first = prepareOrderSubmission(payload, null, createKey)
    const changed = prepareOrderSubmission(
      { items: [{ product_id: 1, quantity: 3 }] },
      first,
      createKey
    )

    expect(changed.idempotencyKey).toBe('second-key')
  })

  // GAP-013 negative regression: once a failed submission is cleared, the very
  // same contents are a NEW order and must carry a NEW key. Before the fix the
  // stale pendingSubmission survived onError, so this path replayed the old
  // key and the backend answered with an idempotent replay instead of a create.
  it('creates a new key when the same payload follows a cleared submission', () => {
    const createKey = vi
      .fn<() => string>()
      .mockReturnValueOnce('first-key')
      .mockReturnValueOnce('retry-key')

    const first = prepareOrderSubmission(payload, null, createKey)
    // Simulates orderApi.create failing -> onError -> setPendingSubmission(null).
    const cleared: PendingOrderSubmission | null = null
    const retried = prepareOrderSubmission(payload, cleared, createKey)

    expect(retried.idempotencyKey).toBe('retry-key')
    expect(retried.idempotencyKey).not.toBe(first.idempotencyKey)
    expect(createKey).toHaveBeenCalledTimes(2)
  })
})

// GAP-012: the order id is a 63-bit Snowflake integer. JSON.parse yields a
// float64, and every integer above 2^53 loses precision, so the id the client
// would send back differs from the id the backend holds -> 404 on
// detail/pay/finish/cancel. String ids keep the digits intact.
describe('order id precision (GAP-012)', () => {
  it('truncates a 19-digit id when coerced to number', () => {
    const id = '1900000000000000001'
    const truncated = String(Number(id))

    expect(truncated).not.toBe(id)
    expect(truncated).toBe('1900000000000000000')
    expect(Number.isSafeInteger(Number(id))).toBe(false)
  })
})
