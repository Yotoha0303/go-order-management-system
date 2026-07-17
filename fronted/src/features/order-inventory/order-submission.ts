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
