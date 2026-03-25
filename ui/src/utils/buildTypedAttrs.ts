import type { SnapshotAttribute } from '../types'

/**
 * Converts raw string attribute values to their proper types based on schema.
 * Number-type attributes are parsed with parseFloat; empty strings are skipped.
 */
export function buildTypedAttrs(
  rawAttrs: Record<string, string>,
  schemaAttrs: SnapshotAttribute[],
): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(rawAttrs)) {
    if (v === '') continue
    const schemaAttr = schemaAttrs.find(a => a.name === k)
    if (schemaAttr?.type === 'number') {
      result[k] = parseFloat(v)
    } else {
      result[k] = v
    }
  }
  return result
}
