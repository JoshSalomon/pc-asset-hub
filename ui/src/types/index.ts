export type Role = 'RO' | 'RW' | 'Admin' | 'SuperAdmin'

export interface EntityType {
  id: string
  name: string
  description?: string
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
  enum_name?: string
  ordinal: number
  required: boolean
  system?: boolean
}

export interface Association {
  id: string
  entity_type_version_id: string
  name: string
  target_entity_type_id: string
  type: 'containment' | 'directional' | 'bidirectional'
  source_role: string
  target_role: string
  source_cardinality: string
  target_cardinality: string
  direction: 'outgoing' | 'incoming'
  source_entity_type_id?: string
}

export interface Enum {
  id: string
  name: string
  description?: string
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
  description?: string
  lifecycle_stage: 'development' | 'testing' | 'production'
  created_at: string
  updated_at: string
}

export interface CatalogVersionPin {
  entity_type_name: string
  entity_type_id: string
  entity_type_version_id: string
  version: number
  description?: string
}

export interface LifecycleTransition {
  id: string
  from_stage: string
  to_stage: string
  performed_by: string
  performed_at: string
  notes?: string
}

export interface RenameEntityTypeResponse {
  entity_type: EntityType
  was_deep_copy: boolean
}

export interface ContainmentTreeNode {
  entity_type: EntityType
  versions: EntityTypeVersion[]
  latest_version: number
  children: ContainmentTreeNode[]
}

export interface SnapshotAttribute {
  id: string
  name: string
  description: string
  type: string
  enum_id?: string
  enum_name?: string
  ordinal: number
  required: boolean
  system?: boolean
}

export interface SnapshotAssociation {
  id: string
  name: string
  type: string
  target_entity_type_id: string
  target_entity_type_name: string
  source_role: string
  target_role: string
  source_cardinality: string
  target_cardinality: string
  direction: 'outgoing' | 'incoming'
  source_entity_type_id?: string
  source_entity_type_name?: string
}

export interface VersionSnapshot {
  entity_type: EntityType
  version: EntityTypeVersion
  attributes: SnapshotAttribute[]
  associations: SnapshotAssociation[]
}

export interface Catalog {
  id: string
  name: string
  description: string
  catalog_version_id: string
  catalog_version_label?: string
  validation_status: 'draft' | 'valid' | 'invalid'
  published: boolean
  published_at?: string
  created_at: string
  updated_at: string
}

export interface EntityInstance {
  id: string
  entity_type_id: string
  catalog_id: string
  parent_instance_id?: string
  name: string
  description: string
  version: number
  attributes: AttributeValueResponse[]
  parent_chain?: ParentChainEntry[]
  created_at: string
  updated_at: string
}

export interface AttributeValueResponse {
  name: string
  type: string
  value: string | number | null
  required?: boolean
  system?: boolean
}

export interface AssociationLink {
  id: string
  association_id: string
  source_instance_id: string
  target_instance_id: string
  created_at: string
}

export interface ReferenceDetail {
  link_id: string
  association_name: string
  association_type: string
  instance_id: string
  instance_name: string
  entity_type_name: string
}

export interface ParentChainEntry {
  instance_id: string
  instance_name: string
  entity_type_name: string
}

export interface TreeNodeResponse {
  instance_id: string
  instance_name: string
  entity_type_name: string
  description: string
  children: TreeNodeResponse[]
}

export interface ValidationError {
  entity_type: string
  instance_name: string
  field: string
  violation: string
}

export interface ValidationResult {
  status: 'valid' | 'invalid'
  errors: ValidationError[]
}

export interface ListResponse<T> {
  items: T[]
  total: number
}
