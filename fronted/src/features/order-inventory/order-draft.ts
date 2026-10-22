import { createRowId } from './order-submission'

/**
 * One editable row of the create-order form.
 *
 * `rowId` exists only for React reconciliation (GAP-013): it is never sent to
 * the backend and never derived from the array index, so a reset produces new
 * identities and the row subtree (including Radix Select) remounts cleanly.
 */
export type DraftOrderItem = {
  rowId: string
  product_id: string
  quantity: string
}

export function createDraftOrderItem(
  generateRowId: () => string = createRowId
): DraftOrderItem {
  return { rowId: generateRowId(), product_id: '', quantity: '1' }
}

export function createInitialDraftItems(): DraftOrderItem[] {
  return [createDraftOrderItem()]
}

export function replaceDraftItem(
  items: DraftOrderItem[],
  index: number,
  patch: Partial<Omit<DraftOrderItem, 'rowId'>>
): DraftOrderItem[] {
  return items.map((item, itemIndex) =>
    itemIndex === index ? { ...item, ...patch } : item
  )
}

export function appendDraftItem(
  items: DraftOrderItem[],
  generateRowId: () => string = createRowId
): DraftOrderItem[] {
  return [...items, createDraftOrderItem(generateRowId)]
}

export function removeDraftItem(
  items: DraftOrderItem[],
  index: number
): DraftOrderItem[] {
  return items.filter((_, itemIndex) => itemIndex !== index)
}

/**
 * Clear the form back to a single blank row.
 *
 * A fresh `rowId` is mandatory here: reusing the old one is exactly the bug
 * this module exists to prevent (the Select would keep its stale display).
 */
export function resetDraftItems(
  generateRowId: () => string = createRowId
): DraftOrderItem[] {
  return [createDraftOrderItem(generateRowId)]
}
