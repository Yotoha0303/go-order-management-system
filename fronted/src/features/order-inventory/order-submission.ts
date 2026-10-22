import type { CreateOrderPayload } from './types'

export type PendingOrderSubmission = {
  fingerprint: string
  idempotencyKey: string
}

function fingerprintOrder(payload: CreateOrderPayload) {
  return JSON.stringify(payload.items)
}

/**
 * Generate an idempotency key that works on plain HTTP hosts too.
 * crypto.randomUUID() is secure-context only (HTTPS or localhost), so cloud
 * demos served over http://public-ip would throw and block order submit.
 */
export function createIdempotencyKey(): string {
  if (
    typeof globalThis.crypto !== 'undefined' &&
    typeof globalThis.crypto.randomUUID === 'function'
  ) {
    try {
      return globalThis.crypto.randomUUID()
    } catch {
      // Insecure context or restricted environment; fall through.
    }
  }

  const randomPart =
    typeof globalThis.crypto !== 'undefined' &&
    typeof globalThis.crypto.getRandomValues === 'function'
      ? Array.from(globalThis.crypto.getRandomValues(new Uint8Array(16)))
          .map((byte) => byte.toString(16).padStart(2, '0'))
          .join('')
      : `${Date.now().toString(36)}${Math.random().toString(36).slice(2, 12)}`

  return `idem-${randomPart}`
}

let rowIdSequence = 0

/**
 * Generate a stable identity for a draft order row (GAP-013).
 *
 * React reconciles lists by key. Using the array index as key means a reset
 * (`key=0` -> `key=0`) does not remount the row, so Radix Select keeps the
 * previously displayed value after the draft is cleared. A per-row identity
 * that changes whenever a row is created forces the remount.
 *
 * The counter guarantees uniqueness within the session; the random part keeps
 * ids distinct across page loads (same degradation strategy as the
 * idempotency key, so plain-HTTP hosts keep working).
 */
export function createRowId(): string {
  rowIdSequence += 1
  return `row-${rowIdSequence}-${createIdempotencyKey()}`
}

/**
 * Decide the idempotency key for one order submission.
 *
 * The fingerprint match means "this is the same submission still in flight",
 * NOT "the user is ordering the same contents again": reuse is only valid
 * while `previous` is the submission currently awaiting a response. Callers
 * MUST clear the pending submission on both success and error, so the next
 * click always starts a brand-new order with a brand-new key (GAP-013).
 */
export function prepareOrderSubmission(
  payload: CreateOrderPayload,
  previous?: PendingOrderSubmission | null,
  createKey: () => string = createIdempotencyKey
): PendingOrderSubmission {
  const fingerprint = fingerprintOrder(payload)
  if (previous?.fingerprint === fingerprint) return previous

  return {
    fingerprint,
    idempotencyKey: createKey(),
  }
}
