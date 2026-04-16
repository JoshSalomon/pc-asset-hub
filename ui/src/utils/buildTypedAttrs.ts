import type { SnapshotAttribute } from '../types'

/**
 * Converts raw string attribute values to their proper types based on schema.
 * Number/integer-type attributes are parsed with parseFloat/parseInt;
 * boolean-type attributes are converted to boolean; empty strings are skipped.
 */
export function buildTypedAttrs(
  rawAttrs: Record<string, string>,
  schemaAttrs: SnapshotAttribute[],
  isEdit = false,
): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(rawAttrs)) {
    if (v === '') {
      // On edit, send null to explicitly clear the value (TD-98).
      // On create, skip empty values (draft mode allows missing attrs).
      if (isEdit) result[k] = null
      continue
    }
    const schemaAttr = schemaAttrs.find(a => a.name === k)
    const baseType = schemaAttr?.base_type
    if (baseType === 'number') {
      result[k] = parseFloat(v)
    } else if (baseType === 'integer') {
      result[k] = parseInt(v, 10)
    } else if (baseType === 'boolean') {
      result[k] = v === 'true'
    } else {
      result[k] = v
    }
  }
  return result
}
