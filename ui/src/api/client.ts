import type {
  EntityType,
  EntityTypeVersion,
  Attribute,
  Association,
  TypeDefinition,
  TypeDefinitionVersion,
  Catalog,
  CatalogVersion,
  CatalogVersionPin,
  LifecycleTransition,
  RenameEntityTypeResponse,
  VersionDiff,
  ContainmentTreeNode,
  VersionSnapshot,
  UpdatePinResponse,
  EntityInstance,
  AssociationLink,
  ReferenceDetail,
  TreeNodeResponse,
  ValidationResult,
  ListResponse,
} from '../types'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/meta/v1'
const DATA_BASE_URL = import.meta.env.VITE_DATA_API_BASE_URL || '/api/data/v1'

let currentRole: string | null = null

export function setAuthRole(role: string | null) {
  currentRole = role
}

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options?.headers as Record<string, string>) || {}),
  }
  if (currentRole) {
    headers['X-User-Role'] = currentRole
  }
  const res = await fetch(url, { ...options, headers })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`${res.status}: ${body}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  entityTypes: {
    list: (params?: { name?: string }) => {
      const query = params?.name ? `?name=${encodeURIComponent(params.name)}` : ''
      return fetchJSON<ListResponse<EntityType>>(`${BASE_URL}/entity-types${query}`)
    },
    get: (id: string) => fetchJSON<EntityType>(`${BASE_URL}/entity-types/${id}`),
    create: (data: { name: string; description?: string }) =>
      fetchJSON<{ entity_type: EntityType }>(`${BASE_URL}/entity-types`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: { description: string }) =>
      fetchJSON(`${BASE_URL}/entity-types/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchJSON(`${BASE_URL}/entity-types/${id}`, { method: 'DELETE' }),
    copy: (id: string, data: { source_version: number; new_name: string }) =>
      fetchJSON(`${BASE_URL}/entity-types/${id}/copy`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    rename: (id: string, name: string, deepCopyAllowed = false) =>
      fetchJSON<RenameEntityTypeResponse>(`${BASE_URL}/entity-types/${id}/rename`, {
        method: 'POST',
        body: JSON.stringify({ name, deep_copy_allowed: deepCopyAllowed }),
      }),
    containmentTree: () =>
      fetchJSON<ContainmentTreeNode[]>(`${BASE_URL}/entity-types/containment-tree`),
  },
  attributes: {
    list: (entityTypeId: string) =>
      fetchJSON<ListResponse<Attribute>>(`${BASE_URL}/entity-types/${entityTypeId}/attributes`),
    add: (entityTypeId: string, data: { name: string; description?: string; type_definition_version_id: string; required?: boolean }) =>
      fetchJSON<EntityTypeVersion>(`${BASE_URL}/entity-types/${entityTypeId}/attributes`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    remove: (entityTypeId: string, name: string) =>
      fetchJSON(`${BASE_URL}/entity-types/${entityTypeId}/attributes/${encodeURIComponent(name)}`, {
        method: 'DELETE',
      }),
    reorder: (entityTypeId: string, orderedIds: string[]) =>
      fetchJSON(`${BASE_URL}/entity-types/${entityTypeId}/attributes/reorder`, {
        method: 'PUT',
        body: JSON.stringify({ ordered_ids: orderedIds }),
      }),
    edit: (entityTypeId: string, name: string, data: { name?: string; description?: string; type_definition_version_id?: string; required?: boolean }) =>
      fetchJSON<EntityTypeVersion>(`${BASE_URL}/entity-types/${entityTypeId}/attributes/${encodeURIComponent(name)}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    copyFrom: (entityTypeId: string, data: { source_entity_type_id: string; source_version: number; attribute_names: string[] }) =>
      fetchJSON<EntityTypeVersion>(`${BASE_URL}/entity-types/${entityTypeId}/attributes/copy`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
  },
  associations: {
    list: (entityTypeId: string) =>
      fetchJSON<ListResponse<Association>>(`${BASE_URL}/entity-types/${entityTypeId}/associations`),
    create: (entityTypeId: string, data: { target_entity_type_id: string; type: string; name: string; source_role?: string; target_role?: string; source_cardinality?: string; target_cardinality?: string }) =>
      fetchJSON<EntityTypeVersion>(`${BASE_URL}/entity-types/${entityTypeId}/associations`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    edit: (entityTypeId: string, name: string, data: { name?: string; type?: string; source_role?: string; target_role?: string; source_cardinality?: string; target_cardinality?: string }) =>
      fetchJSON<EntityTypeVersion>(`${BASE_URL}/entity-types/${entityTypeId}/associations/${encodeURIComponent(name)}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (entityTypeId: string, name: string) =>
      fetchJSON(`${BASE_URL}/entity-types/${entityTypeId}/associations/${encodeURIComponent(name)}`, {
        method: 'DELETE',
      }),
  },
  typeDefinitions: {
    list: (params?: { base_type?: string }) => {
      const query = params?.base_type ? `?base_type=${encodeURIComponent(params.base_type)}` : ''
      return fetchJSON<ListResponse<TypeDefinition>>(`${BASE_URL}/type-definitions${query}`)
    },
    get: (id: string) => fetchJSON<TypeDefinition>(`${BASE_URL}/type-definitions/${id}`),
    create: (data: { name: string; description?: string; base_type: string; constraints?: Record<string, unknown> }) =>
      fetchJSON<TypeDefinition>(`${BASE_URL}/type-definitions`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: { description?: string; constraints?: Record<string, unknown> }) =>
      fetchJSON<TypeDefinitionVersion>(`${BASE_URL}/type-definitions/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchJSON(`${BASE_URL}/type-definitions/${id}`, { method: 'DELETE' }),
    listVersions: (id: string) =>
      fetchJSON<ListResponse<TypeDefinitionVersion>>(`${BASE_URL}/type-definitions/${id}/versions`),
  },
  versions: {
    list: (entityTypeId: string) =>
      fetchJSON<ListResponse<EntityTypeVersion>>(`${BASE_URL}/entity-types/${entityTypeId}/versions`),
    diff: (entityTypeId: string, v1: number, v2: number) =>
      fetchJSON<VersionDiff>(`${BASE_URL}/entity-types/${entityTypeId}/versions/diff?v1=${v1}&v2=${v2}`),
    snapshot: (entityTypeId: string, version: number) =>
      fetchJSON<VersionSnapshot>(`${BASE_URL}/entity-types/${entityTypeId}/versions/${version}/snapshot`),
  },
  catalogVersions: {
    list: (params?: { stage?: string }) => {
      const query = params?.stage ? `?stage=${encodeURIComponent(params.stage)}` : ''
      return fetchJSON<ListResponse<CatalogVersion>>(`${BASE_URL}/catalog-versions${query}`)
    },
    get: (id: string) => fetchJSON<CatalogVersion>(`${BASE_URL}/catalog-versions/${id}`),
    listPins: (id: string) =>
      fetchJSON<ListResponse<CatalogVersionPin>>(`${BASE_URL}/catalog-versions/${id}/pins`),
    listTransitions: (id: string) =>
      fetchJSON<ListResponse<LifecycleTransition>>(`${BASE_URL}/catalog-versions/${id}/transitions`),
    create: (data: { version_label: string; description?: string; pins?: { entity_type_version_id: string }[] }) =>
      fetchJSON<CatalogVersion>(`${BASE_URL}/catalog-versions`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    promote: (id: string) =>
      fetchJSON(`${BASE_URL}/catalog-versions/${id}/promote`, { method: 'POST' }),
    demote: (id: string, targetStage: string) =>
      fetchJSON(`${BASE_URL}/catalog-versions/${id}/demote`, {
        method: 'POST',
        body: JSON.stringify({ target_stage: targetStage }),
      }),
    delete: (id: string) =>
      fetchJSON(`${BASE_URL}/catalog-versions/${id}`, { method: 'DELETE' }),
    update: (id: string, data: { version_label?: string; description?: string }) =>
      fetchJSON<CatalogVersion>(`${BASE_URL}/catalog-versions/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    addPin: (id: string, entityTypeVersionId: string) =>
      fetchJSON(`${BASE_URL}/catalog-versions/${id}/pins`, {
        method: 'POST',
        body: JSON.stringify({ entity_type_version_id: entityTypeVersionId }),
      }),
    updatePin: (id: string, pinId: string, entityTypeVersionId: string) =>
      fetchJSON<UpdatePinResponse>(`${BASE_URL}/catalog-versions/${id}/pins/${pinId}`, {
        method: 'PUT',
        body: JSON.stringify({ entity_type_version_id: entityTypeVersionId }),
      }),
    updatePinDryRun: (id: string, pinId: string, entityTypeVersionId: string) =>
      fetchJSON<UpdatePinResponse>(`${BASE_URL}/catalog-versions/${id}/pins/${pinId}?dry_run=true`, {
        method: 'PUT',
        body: JSON.stringify({ entity_type_version_id: entityTypeVersionId }),
      }),
    removePin: (id: string, pinId: string) =>
      fetchJSON<void>(`${BASE_URL}/catalog-versions/${id}/pins/${pinId}`, { method: 'DELETE' }),
  },

  catalogs: {
    list: (params?: { catalog_version_id?: string; validation_status?: string }) => {
      const query = new URLSearchParams()
      if (params?.catalog_version_id) query.set('catalog_version_id', params.catalog_version_id)
      if (params?.validation_status) query.set('validation_status', params.validation_status)
      const qs = query.toString()
      return fetchJSON<ListResponse<Catalog>>(`${DATA_BASE_URL}/catalogs${qs ? `?${qs}` : ''}`)
    },
    get: (name: string) =>
      fetchJSON<Catalog>(`${DATA_BASE_URL}/catalogs/${name}`),
    create: (data: { name: string; description?: string; catalog_version_id: string }) =>
      fetchJSON<Catalog>(`${DATA_BASE_URL}/catalogs`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    delete: (name: string) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${name}`, { method: 'DELETE' }),
    update: (name: string, data: { name?: string; description?: string; catalog_version_id?: string }) =>
      fetchJSON<Catalog>(`${DATA_BASE_URL}/catalogs/${name}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    validate: (name: string) =>
      fetchJSON<ValidationResult>(`${DATA_BASE_URL}/catalogs/${name}/validate`, { method: 'POST' }),
    publish: (name: string) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${name}/publish`, { method: 'POST' }),
    unpublish: (name: string) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${name}/unpublish`, { method: 'POST' }),
    copy: (data: { source: string; name: string; description?: string }) =>
      fetchJSON<Catalog>(`${DATA_BASE_URL}/catalogs/copy`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    replace: (data: { source: string; target: string; archive_name?: string }) =>
      fetchJSON<Catalog>(`${DATA_BASE_URL}/catalogs/replace`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    export: async (name: string, params?: { entities?: string; source_system?: string }) => {
      const query = new URLSearchParams()
      if (params?.entities) query.set('entities', params.entities)
      if (params?.source_system) query.set('source_system', params.source_system)
      const qs = query.toString()
      const headers: Record<string, string> = { 'Content-Type': 'application/json' }
      if (currentRole) headers['X-User-Role'] = currentRole
      const res = await fetch(`${DATA_BASE_URL}/catalogs/${name}/export${qs ? `?${qs}` : ''}`, { headers })
      if (!res.ok) {
        const body = await res.text()
        throw new Error(`${res.status}: ${body}`)
      }
      return res.json()
    },
    import: (data: unknown, params?: { dry_run?: boolean }) => {
      const query = params?.dry_run ? '?dry_run=true' : ''
      return fetchJSON<unknown>(`${DATA_BASE_URL}/catalogs/import${query}`, {
        method: 'POST',
        body: JSON.stringify(data),
      })
    },
  },

  instances: {
    list: (catalogName: string, entityTypeName: string, params?: { limit?: number; offset?: number; sort?: string; filters?: Record<string, string> }) => {
      const query = new URLSearchParams()
      if (params?.limit !== undefined) query.set('limit', String(params.limit))
      if (params?.offset !== undefined) query.set('offset', String(params.offset))
      if (params?.sort) query.set('sort', params.sort)
      if (params?.filters) {
        for (const [k, v] of Object.entries(params.filters)) {
          query.set(`filter.${k}`, v)
        }
      }
      const qs = query.toString()
      return fetchJSON<ListResponse<EntityInstance>>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityTypeName}${qs ? `?${qs}` : ''}`)
    },
    tree: (catalogName: string) =>
      fetchJSON<TreeNodeResponse[]>(`${DATA_BASE_URL}/catalogs/${catalogName}/tree`),
    get: (catalogName: string, entityTypeName: string, instanceId: string) =>
      fetchJSON<EntityInstance>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityTypeName}/${instanceId}`),
    create: (catalogName: string, entityTypeName: string, data: { name: string; description?: string; attributes?: Record<string, unknown> }) =>
      fetchJSON<EntityInstance>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityTypeName}`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (catalogName: string, entityTypeName: string, instanceId: string, data: { version: number; name?: string; description?: string; attributes?: Record<string, unknown> }) =>
      fetchJSON<EntityInstance>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityTypeName}/${instanceId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (catalogName: string, entityTypeName: string, instanceId: string) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityTypeName}/${instanceId}`, { method: 'DELETE' }),
    setParent: (catalogName: string, entityType: string, instanceId: string, data: { parent_type: string; parent_instance_id: string }) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityType}/${instanceId}/parent`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    createContained: (catalogName: string, parentType: string, parentId: string, childType: string, data: { name: string; description?: string; attributes?: Record<string, unknown> }) =>
      fetchJSON<EntityInstance>(`${DATA_BASE_URL}/catalogs/${catalogName}/${parentType}/${parentId}/${childType}`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    listContained: (catalogName: string, parentType: string, parentId: string, childType: string) =>
      fetchJSON<ListResponse<EntityInstance>>(`${DATA_BASE_URL}/catalogs/${catalogName}/${parentType}/${parentId}/${childType}`),
  },

  links: {
    create: (catalogName: string, entityType: string, instanceId: string, data: { target_instance_id: string; association_name: string }) =>
      fetchJSON<AssociationLink>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityType}/${instanceId}/links`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    delete: (catalogName: string, entityType: string, instanceId: string, linkId: string) =>
      fetchJSON(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityType}/${instanceId}/links/${linkId}`, { method: 'DELETE' }),
    forwardRefs: (catalogName: string, entityType: string, instanceId: string) =>
      fetchJSON<ReferenceDetail[]>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityType}/${instanceId}/references`),
    reverseRefs: (catalogName: string, entityType: string, instanceId: string) =>
      fetchJSON<ReferenceDetail[]>(`${DATA_BASE_URL}/catalogs/${catalogName}/${entityType}/${instanceId}/referenced-by`),
  },
}
