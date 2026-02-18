export type Role = 'RO' | 'RW' | 'Admin' | 'SuperAdmin'

export interface EntityType {
  id: string
  name: string
  created_at: string
  updated_at: string
}

export interface EntityTypeVersion {
  id: string
  entity_type_id: string
  version: number
  description: string
  created_at: string
}

export interface Attribute {
  id: string
  name: string
  description: string
  type: 'string' | 'number' | 'enum'
  enum_id?: string
  ordinal: number
  required: boolean
}

export interface Association {
  id: string
  entity_type_version_id: string
  target_entity_type_id: string
  type: 'containment' | 'directional' | 'bidirectional'
  source_role: string
  target_role: string
}

export interface Enum {
  id: string
  name: string
  created_at: string
  updated_at: string
}

export interface EnumValue {
  id: string
  value: string
  ordinal: number
}

export interface VersionDiffItem {
  name: string
  change_type: string
  category: string
  old_value?: string
  new_value?: string
}

export interface VersionDiff {
  from_version: number
  to_version: number
  changes: VersionDiffItem[]
}

export interface CatalogVersion {
  id: string
  version_label: string
  lifecycle_stage: 'development' | 'testing' | 'production'
  created_at: string
  updated_at: string
}

export interface ListResponse<T> {
  items: T[]
  total: number
}
