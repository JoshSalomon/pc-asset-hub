import { api } from '../api/client'
import type { CatalogVersionPin, SnapshotAttribute, SnapshotAssociation } from '../types'

export interface SchemaSnapshot {
  attrs: SnapshotAttribute[]
  assocs: SnapshotAssociation[]
  enums: Record<string, string[]>
}

export async function loadSchemaSnapshot(
  pins: CatalogVersionPin[],
  typeName: string,
): Promise<SchemaSnapshot> {
  const pin = pins.find(p => p.entity_type_name === typeName)
  if (!pin) return { attrs: [], assocs: [], enums: {} }
  const snapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
  const attrs = snapshot.attributes || []
  const assocs = snapshot.associations || []
  const enums: Record<string, string[]> = {}
  for (const attr of attrs) {
    if (attr.base_type === 'enum' && attr.type_definition_version_id) {
      const vals = (attr.constraints?.values as string[]) || []
      if (vals.length > 0) enums[attr.type_definition_version_id] = vals
    }
  }
  return { attrs, assocs, enums }
}
