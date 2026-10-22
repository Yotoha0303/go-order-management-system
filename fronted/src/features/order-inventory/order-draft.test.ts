import { describe, expect, it } from 'vitest'
import {
  appendDraftItem,
  createDraftOrderItem,
  createInitialDraftItems,
  removeDraftItem,
  replaceDraftItem,
  resetDraftItems,
  type DraftOrderItem,
} from './order-draft'

/** Deterministic row id generator so the assertions never depend on random. */
function countingRowIds() {
  let next = 0
  return () => `row-${++next}`
}

describe('order draft rows', () => {
  it('starts with a single blank row', () => {
    const items = createInitialDraftItems()

    expect(items).toHaveLength(1)
    expect(items[0].product_id).toBe('')
    expect(items[0].quantity).toBe('1')
    expect(items[0].rowId).not.toBe('')
  })

  it('mints a distinct rowId for every appended row', () => {
    const generateRowId = countingRowIds()
    const first = createDraftOrderItem(generateRowId)
    const items = appendDraftItem([first], generateRowId)

    expect(items).toHaveLength(2)
    expect(items[0].rowId).not.toBe(items[1].rowId)
  })

  it('keeps rowId stable when editing a row', () => {
    const items = [createDraftOrderItem(() => 'row-a')]
    const edited = replaceDraftItem(items, 0, { product_id: '7' })

    expect(edited[0].rowId).toBe('row-a')
    expect(edited[0].product_id).toBe('7')
  })

  it('does not mutate the input array on append or remove', () => {
    const items = [createDraftOrderItem(() => 'row-a')]
    const appended = appendDraftItem(items, () => 'row-b')
    const removed = removeDraftItem(appended, 0)

    expect(items).toHaveLength(1)
    expect(appended).toHaveLength(2)
    expect(removed).toHaveLength(1)
    expect(removed[0].rowId).toBe('row-b')
  })

  // GAP-013 negative regression: the reset used to keep key=0, so React reused
  // the same row instance and Radix Select kept displaying the previous
  // product. Every rowId must change, otherwise the bug is back.
  it('reset changes every rowId so the row subtree remounts', () => {
    const generateRowId = countingRowIds()
    const before: DraftOrderItem[] = [
      createDraftOrderItem(generateRowId),
      createDraftOrderItem(generateRowId),
    ]
    const after = resetDraftItems(generateRowId)

    expect(after).toHaveLength(1)
    expect(after[0].product_id).toBe('')
    expect(after[0].quantity).toBe('1')

    const previousIds = new Set(before.map((item) => item.rowId))
    expect(previousIds.has(after[0].rowId)).toBe(false)
  })

  it('reset called twice never returns the same identity', () => {
    const generateRowId = countingRowIds()
    const first = resetDraftItems(generateRowId)
    const second = resetDraftItems(generateRowId)

    expect(first[0].rowId).not.toBe(second[0].rowId)
  })
})
