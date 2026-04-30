// Tests for the API client module itself — no mocking.
// Exercises fetchJSON, setAuthRole, and all api methods.
// Uses a fake fetch to avoid needing a real server.

import { expect, test, vi, beforeEach } from 'vitest'
import { setAuthRole, api } from './client'

// Intercept fetch at the global level (not mocking the module)
const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch)
  mockFetch.mockReset()
  setAuthRole(null)
})

function jsonResponse(data: object, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  })
}

function noContentResponse() {
  return Promise.resolve({
    ok: true,
    status: 204,
    json: () => Promise.reject(new Error('no content')),
    text: () => Promise.resolve(''),
  })
}

test('setAuthRole adds X-User-Role header to requests', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  setAuthRole('Admin')
  await api.entityTypes.list()

  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types'),
    expect.objectContaining({
      headers: expect.objectContaining({ 'X-User-Role': 'Admin' }),
    }),
  )
})

test('setAuthRole(null) omits X-User-Role header', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  setAuthRole(null)
  await api.entityTypes.list()

  const headers = mockFetch.mock.calls[0][1].headers
  expect(headers['X-User-Role']).toBeUndefined()
})

test('entityTypes.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.entityTypes.list()
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types'),
    expect.anything(),
  )
})

test('entityTypes.list with name param adds query string', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.entityTypes.list({ name: 'Model' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('?name=Model'),
    expect.anything(),
  )
})

test('entityTypes.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'et-1', name: 'Test' }))

  const result = await api.entityTypes.get('et-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1'),
    expect.anything(),
  )
  expect(result.name).toBe('Test')
})

test('entityTypes.create sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ entity_type: { id: 'et-1' } }))

  await api.entityTypes.create({ name: 'New', description: 'desc' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'New', description: 'desc' }),
    }),
  )
})

test('entityTypes.update sends PUT', async () => {
  mockFetch.mockReturnValue(jsonResponse({}))

  await api.entityTypes.update('et-1', { description: 'updated' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1'),
    expect.objectContaining({ method: 'PUT' }),
  )
})

test('entityTypes.delete sends DELETE and handles 204', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  const result = await api.entityTypes.delete('et-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
  expect(result).toBeUndefined()
})

test('entityTypes.copy sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ entity_type: { id: 'et-2' } }))

  await api.entityTypes.copy('et-1', { source_version: 1, new_name: 'Copy' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/copy'),
    expect.objectContaining({ method: 'POST' }),
  )
})

test('catalogVersions.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.catalogVersions.list()
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions'),
    expect.anything(),
  )
})

test('catalogVersions.create sends POST', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'cv-1' }))

  await api.catalogVersions.create({ version_label: 'v1.0' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions'),
    expect.objectContaining({ method: 'POST' }),
  )
})

test('catalogVersions.promote sends POST', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'promoted' }))

  await api.catalogVersions.promote('cv-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions/cv-1/promote'),
    expect.objectContaining({ method: 'POST' }),
  )
})

test('catalogVersions.demote sends POST with target_stage', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'demoted' }))

  await api.catalogVersions.demote('cv-1', 'development')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions/cv-1/demote'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ target_stage: 'development' }),
    }),
  )
})

test('catalogVersions.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.catalogVersions.delete('cv-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions/cv-1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

test('throws on non-ok response', async () => {
  mockFetch.mockReturnValue(
    Promise.resolve({
      ok: false,
      status: 409,
      text: () => Promise.resolve('conflict error'),
    }),
  )

  await expect(api.entityTypes.create({ name: 'Dup' })).rejects.toThrow('409: conflict error')
})

test('catalogVersions.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'cv-1', version_label: 'v1.0' }))

  const result = await api.catalogVersions.get('cv-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalog-versions/cv-1'),
    expect.anything(),
  )
  expect(result.version_label).toBe('v1.0')
})

// === Attributes ===

test('attributes.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.attributes.list('et-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes'),
    expect.anything(),
  )
})

test('attributes.add sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'v2', version: 2 }))

  await api.attributes.add('et-1', { name: 'host', type_definition_version_id: 'tdv-string', description: 'desc' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'host', type_definition_version_id: 'tdv-string', description: 'desc' }),
    }),
  )
})

test('attributes.add with type_definition_version_id', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'v2', version: 2 }))

  await api.attributes.add('et-1', { name: 'status', type_definition_version_id: 'tdv-enum1' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes'),
    expect.objectContaining({
      body: JSON.stringify({ name: 'status', type_definition_version_id: 'tdv-enum1' }),
    }),
  )
})

test('attributes.remove sends DELETE with encoded name', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.attributes.remove('et-1', 'host name')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes/host%20name'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

test('attributes.reorder sends PUT with ordered_ids', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'reordered' }))

  await api.attributes.reorder('et-1', ['a2', 'a1'])
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes/reorder'),
    expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ ordered_ids: ['a2', 'a1'] }),
    }),
  )
})

// === Associations ===

test('associations.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.associations.list('et-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/associations'),
    expect.anything(),
  )
})

test('associations.create sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'v2', version: 2 }))

  await api.associations.create('et-1', {
    target_entity_type_id: 'et-2',
    type: 'containment',
    name: 'tools',
    source_role: 'parent',
    target_role: 'child',
  })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/associations'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({
        target_entity_type_id: 'et-2',
        type: 'containment',
        name: 'tools',
        source_role: 'parent',
        target_role: 'child',
      }),
    }),
  )
})

test('associations.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.associations.delete('et-1', 'assoc-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/associations/assoc-1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

// === Type Definitions ===

test('typeDefinitions.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.typeDefinitions.list()
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions'),
    expect.anything(),
  )
})

test('typeDefinitions.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'td1', name: 'Status', base_type: 'enum', system: false, latest_version: 1 }))

  const result = await api.typeDefinitions.get('td1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions/td1'),
    expect.anything(),
  )
  expect(result.name).toBe('Status')
})

test('typeDefinitions.create sends POST with name, base_type, and constraints', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'td1', name: 'Status', base_type: 'enum' }))

  await api.typeDefinitions.create({ name: 'Status', base_type: 'enum', constraints: { values: ['active', 'inactive'] } })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'Status', base_type: 'enum', constraints: { values: ['active', 'inactive'] } }),
    }),
  )
})

test('typeDefinitions.update sends PUT with constraints', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'tdv1', version_number: 2 }))

  await api.typeDefinitions.update('td1', { constraints: { values: ['active', 'inactive', 'pending'] } })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions/td1'),
    expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ constraints: { values: ['active', 'inactive', 'pending'] } }),
    }),
  )
})

test('typeDefinitions.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.typeDefinitions.delete('td1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions/td1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

test('typeDefinitions.listVersions calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.typeDefinitions.listVersions('td1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/type-definitions/td1/versions'),
    expect.anything(),
  )
})

// === Versions ===

test('versions.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.versions.list('et-1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/versions'),
    expect.anything(),
  )
})

test('versions.diff calls correct URL with query params', async () => {
  mockFetch.mockReturnValue(jsonResponse({ from_version: 1, to_version: 2, changes: [] }))

  const result = await api.versions.diff('et-1', 1, 2)
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/versions/diff?v1=1&v2=2'),
    expect.anything(),
  )
  expect(result.from_version).toBe(1)
  expect(result.to_version).toBe(2)
})

// === Operational API: Catalogs ===

test('catalogs.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))
  await api.catalogs.list()
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs'),
    expect.anything(),
  )
})

test('catalogs.list with filters passes query params', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))
  await api.catalogs.list({ catalog_version_id: 'cv1', validation_status: 'valid' })
  const url = mockFetch.mock.calls[0][0]
  expect(url).toContain('catalog_version_id=cv1')
  expect(url).toContain('validation_status=valid')
})

test('catalogs.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'c1', name: 'my-cat' }))
  const result = await api.catalogs.get('my-cat')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat'),
    expect.anything(),
  )
  expect(result.name).toBe('my-cat')
})

test('catalogs.create sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'c1', name: 'new-cat' }))
  await api.catalogs.create({ name: 'new-cat', catalog_version_id: 'cv1' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs')
  expect(opts.method).toBe('POST')
  expect(JSON.parse(opts.body)).toEqual({ name: 'new-cat', catalog_version_id: 'cv1' })
})

test('catalogs.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())
  await api.catalogs.delete('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat')
  expect(opts.method).toBe('DELETE')
})

test('catalogs.validate sends POST to /validate', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'valid', errors: [] }))
  const result = await api.catalogs.validate('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/validate')
  expect(opts.method).toBe('POST')
  expect(result.status).toBe('valid')
})

test('catalogs.publish sends POST to /publish', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'published' }))
  await api.catalogs.publish('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/publish')
  expect(opts.method).toBe('POST')
})

test('catalogs.unpublish sends POST to /unpublish', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'unpublished' }))
  await api.catalogs.unpublish('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/unpublish')
  expect(opts.method).toBe('POST')
})

// === Operational API: Instances ===

test('instances.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))
  await api.instances.list('my-cat', 'server')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/server'),
    expect.anything(),
  )
})

test('instances.list with filters passes query params', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))
  await api.instances.list('my-cat', 'server', { limit: 10, offset: 5, sort: 'name:asc', filters: { hostname: 'web' } })
  const url = mockFetch.mock.calls[0][0]
  expect(url).toContain('limit=10')
  expect(url).toContain('offset=5')
  expect(url).toContain('sort=name%3Aasc')
  expect(url).toContain('filter.hostname=web')
})

test('instances.tree calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse([]))
  await api.instances.tree('my-cat')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/tree'),
    expect.anything(),
  )
})

test('instances.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'i1', name: 'server-1' }))
  await api.instances.get('my-cat', 'server', 'i1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/server/i1'),
    expect.anything(),
  )
})

test('instances.create sends POST with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'i1' }))
  await api.instances.create('my-cat', 'server', { name: 'srv-1' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server')
  expect(opts.method).toBe('POST')
})

test('instances.update sends PUT', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'i1', version: 2 }))
  await api.instances.update('my-cat', 'server', 'i1', { version: 1, name: 'new-name' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server/i1')
  expect(opts.method).toBe('PUT')
})

test('instances.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())
  await api.instances.delete('my-cat', 'server', 'i1')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server/i1')
  expect(opts.method).toBe('DELETE')
})

test('instances.setParent sends PUT to /parent', async () => {
  mockFetch.mockReturnValue(jsonResponse({}))
  await api.instances.setParent('my-cat', 'tool', 'i1', { parent_type: 'server', parent_instance_id: 'p1' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/tool/i1/parent')
  expect(opts.method).toBe('PUT')
})

test('instances.createContained sends POST to parent/child route', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'c1' }))
  await api.instances.createContained('my-cat', 'server', 'p1', 'tool', { name: 'tool-1' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server/p1/tool')
  expect(opts.method).toBe('POST')
})

test('instances.listContained calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))
  await api.instances.listContained('my-cat', 'server', 'p1', 'tool')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/server/p1/tool'),
    expect.anything(),
  )
})

// === Operational API: Links ===

test('links.create sends POST', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'link1' }))
  await api.links.create('my-cat', 'server', 'i1', { target_instance_id: 'i2', association_name: 'uses' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server/i1/links')
  expect(opts.method).toBe('POST')
})

test('links.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())
  await api.links.delete('my-cat', 'server', 'i1', 'link1')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/server/i1/links/link1')
  expect(opts.method).toBe('DELETE')
})

test('links.forwardRefs calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse([]))
  await api.links.forwardRefs('my-cat', 'server', 'i1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/server/i1/references'),
    expect.anything(),
  )
})

test('links.reverseRefs calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse([]))
  await api.links.reverseRefs('my-cat', 'server', 'i1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/my-cat/server/i1/referenced-by'),
    expect.anything(),
  )
})

// T-17.87: copyCatalog client function sends POST with correct body
test('catalogs.copy calls correct URL and body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'new-id', name: 'target' }))
  await api.catalogs.copy({ source: 'source', name: 'target', description: 'desc' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/copy'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ source: 'source', name: 'target', description: 'desc' }),
    }),
  )
})

// T-17.88: replaceCatalog client function sends POST with correct body
test('catalogs.replace calls correct URL and body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'src-id', name: 'prod' }))
  await api.catalogs.replace({ source: 'staging', target: 'prod', archive_name: 'prod-archive' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/catalogs/replace'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ source: 'staging', target: 'prod', archive_name: 'prod-archive' }),
    }),
  )
})

// T-18.45: getSnapshot client function returns system attrs in response
test('T-18.45: versions.snapshot returns system attrs', async () => {
  const snapshotData = {
    entity_type: { id: 'et-1', name: 'Model' },
    version: { id: 'v1', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'Name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'Description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a1', name: 'hostname', type: 'string', ordinal: 0, required: false },
    ],
    associations: [],
  }
  mockFetch.mockReturnValue(jsonResponse(snapshotData))

  const result = await api.versions.snapshot('et-1', 1)
  expect(result.attributes).toHaveLength(3)
  expect(result.attributes[0].system).toBe(true)
  expect(result.attributes[0].name).toBe('Name')
  expect(result.attributes[1].system).toBe(true)
  expect(result.attributes[1].name).toBe('Description')
  expect(result.attributes[2].system).toBeUndefined()
})

// === Phase 2 CRUD: catalogVersions.update ===

test('catalogVersions.update sends PUT with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'cv-1', version_label: 'v2.0', description: 'updated' }))

  const result = await api.catalogVersions.update('cv-1', { version_label: 'v2.0', description: 'updated' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalog-versions/cv-1')
  expect(opts.method).toBe('PUT')
  expect(JSON.parse(opts.body)).toEqual({ version_label: 'v2.0', description: 'updated' })
  expect(result.version_label).toBe('v2.0')
})

// === Phase 2 CRUD: catalogVersions.addPin ===

test('catalogVersions.addPin sends POST with entity_type_version_id', async () => {
  mockFetch.mockReturnValue(jsonResponse({ entity_type_version_id: 'etv-1' }, 201))

  await api.catalogVersions.addPin('cv-1', 'etv-1')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalog-versions/cv-1/pins')
  expect(opts.method).toBe('POST')
  expect(JSON.parse(opts.body)).toEqual({ entity_type_version_id: 'etv-1' })
})

// === T-28.21: catalogVersions.updatePin ===

test('T-28.21: catalogVersions.updatePin sends PUT with entity_type_version_id', async () => {
  mockFetch.mockReturnValue(jsonResponse({ pin_id: 'pin-1', entity_type_version_id: 'etv-new' }))

  await api.catalogVersions.updatePin('cv-1', 'pin-1', 'etv-new')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalog-versions/cv-1/pins/pin-1')
  expect(opts.method).toBe('PUT')
  expect(JSON.parse(opts.body)).toEqual({ entity_type_version_id: 'etv-new' })
})

test('catalogVersions.updatePinDryRun sends PUT with dry_run=true', async () => {
  mockFetch.mockReturnValue(jsonResponse({ pin: {}, migration: { affected_instances: 5 } }))

  await api.catalogVersions.updatePinDryRun('cv-1', 'pin-1', 'etv-new')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalog-versions/cv-1/pins/pin-1?dry_run=true')
  expect(opts.method).toBe('PUT')
  expect(JSON.parse(opts.body)).toEqual({ entity_type_version_id: 'etv-new' })
})

// === Phase 2 CRUD: catalogVersions.removePin ===

test('catalogVersions.removePin sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.catalogVersions.removePin('cv-1', 'pin-1')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalog-versions/cv-1/pins/pin-1')
  expect(opts.method).toBe('DELETE')
})

// === Phase 2 CRUD: catalogs.update ===

test('catalogs.update sends PUT with body', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'c1', name: 'my-cat', description: 'new desc' }))

  const result = await api.catalogs.update('my-cat', { description: 'new desc', catalog_version_id: 'cv-2' })
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat')
  expect(opts.method).toBe('PUT')
  expect(JSON.parse(opts.body)).toEqual({ description: 'new desc', catalog_version_id: 'cv-2' })
  expect(result.description).toBe('new desc')
})

// T-18.46: listAttributes client function returns system attrs in response
test('T-18.46: attributes.list returns system attrs', async () => {
  const attrsData = {
    items: [
      { id: 'sys-name', name: 'Name', description: 'Instance name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'Description', description: 'Instance description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a1', name: 'hostname', description: '', type: 'string', ordinal: 0, required: false },
    ],
    total: 3,
  }
  mockFetch.mockReturnValue(jsonResponse(attrsData))

  const result = await api.attributes.list('et-1')
  expect(result.items).toHaveLength(3)
  expect(result.items[0].system).toBe(true)
  expect(result.items[0].name).toBe('Name')
  expect(result.items[1].system).toBe(true)
  expect(result.items[2].system).toBeUndefined()
})

// === Catalog Export/Import ===

test('catalogs.export calls correct URL and returns data', async () => {
  const exportData = { catalog: { name: 'my-cat' }, entity_types: [] }
  mockFetch.mockReturnValue(Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve(exportData),
    text: () => Promise.resolve(JSON.stringify(exportData)),
  }))

  const result = await api.catalogs.export('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/export')
  expect(opts.headers).toHaveProperty('Content-Type', 'application/json')
  expect(result).toEqual(exportData)
})

test('catalogs.export with query params', async () => {
  const exportData = { catalog: { name: 'my-cat' }, entity_types: [] }
  mockFetch.mockReturnValue(Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve(exportData),
    text: () => Promise.resolve(JSON.stringify(exportData)),
  }))

  await api.catalogs.export('my-cat', { entities: 'all', source_system: 'test' })
  const url = mockFetch.mock.calls[0][0]
  expect(url).toContain('entities=all')
  expect(url).toContain('source_system=test')
})

test('catalogs.export includes auth role header', async () => {
  const exportData = { catalog: { name: 'my-cat' } }
  mockFetch.mockReturnValue(Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve(exportData),
    text: () => Promise.resolve(JSON.stringify(exportData)),
  }))

  setAuthRole('Admin')
  await api.catalogs.export('my-cat')
  const headers = mockFetch.mock.calls[0][1].headers
  expect(headers['X-User-Role']).toBe('Admin')
})

test('catalogs.export throws on error', async () => {
  mockFetch.mockReturnValue(Promise.resolve({
    ok: false,
    status: 403,
    text: () => Promise.resolve('forbidden'),
  }))

  await expect(api.catalogs.export('my-cat')).rejects.toThrow('403: forbidden')
})

test('catalogs.import sends POST with body', async () => {
  const importData = { data: { catalog: { name: 'imported' } } }
  mockFetch.mockReturnValue(jsonResponse({ status: 'ok', catalog_name: 'imported' }))

  await api.catalogs.import(importData)
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/import')
  expect(url).not.toContain('dry_run')
  expect(opts.method).toBe('POST')
  expect(JSON.parse(opts.body)).toEqual(importData)
})

test('catalogs.import with dry_run sends query param', async () => {
  const importData = { data: { catalog: { name: 'imported' } } }
  mockFetch.mockReturnValue(jsonResponse({ status: 'ok', collisions: [] }))

  await api.catalogs.import(importData, { dry_run: true })
  const url = mockFetch.mock.calls[0][0]
  expect(url).toContain('?dry_run=true')
})
