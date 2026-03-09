import type {
  EntityType,
  EntityTypeVersion,
  Attribute,
  Association,
  Enum,
  EnumValue,
  Catalog,
  CatalogVersion,
  CatalogVersionPin,
  LifecycleTransition,
  RenameEntityTypeResponse,
  VersionDiff,
  ContainmentTreeNode,
  VersionSnapshot,
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
    add: (entityTypeId: string, data: { name: string; description?: string; type: string; enum_id?: string; required?: boolean }) =>
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
    edit: (entityTypeId: string, name: string, data: { name?: string; description?: string; type?: string; enum_id?: string; required?: boolean }) =>
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
  enums: {
    list: () => fetchJSON<ListResponse<Enum>>(`${BASE_URL}/enums`),
    get: (id: string) => fetchJSON<Enum>(`${BASE_URL}/enums/${id}`),
    create: (data: { name: string; values?: string[] }) =>
      fetchJSON<Enum>(`${BASE_URL}/enums`, {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: { name: string }) =>
      fetchJSON(`${BASE_URL}/enums/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) =>
      fetchJSON(`${BASE_URL}/enums/${id}`, { method: 'DELETE' }),
    listValues: (enumId: string) =>
      fetchJSON<ListResponse<EnumValue>>(`${BASE_URL}/enums/${enumId}/values`),
    addValue: (enumId: string, value: string) =>
      fetchJSON(`${BASE_URL}/enums/${enumId}/values`, {
        method: 'POST',
        body: JSON.stringify({ value }),
      }),
    removeValue: (enumId: string, valueId: string) =>
      fetchJSON(`${BASE_URL}/enums/${enumId}/values/${valueId}`, {
        method: 'DELETE',
      }),
    reorderValues: (enumId: string, orderedIds: string[]) =>
      fetchJSON(`${BASE_URL}/enums/${enumId}/values/reorder`, {
        method: 'PUT',
        body: JSON.stringify({ ordered_ids: orderedIds }),
      }),
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
    create: (data: { version_label: string; pins?: { entity_type_version_id: string }[] }) =>
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
  },
}
