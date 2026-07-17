import { describe, expect, it, vi } from 'vitest'
import {
  createIdempotencyKey,
  prepareOrderSubmission,
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

describe('prepareOrderSubmission', () => {
  it('reuses the key when an unchanged request is retried', () => {
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
})
