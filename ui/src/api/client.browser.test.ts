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

  await api.attributes.add('et-1', { name: 'host', type: 'string', description: 'desc' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'host', type: 'string', description: 'desc' }),
    }),
  )
})

test('attributes.add with enum_id', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'v2', version: 2 }))

  await api.attributes.add('et-1', { name: 'status', type: 'enum', enum_id: 'e1' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/entity-types/et-1/attributes'),
    expect.objectContaining({
      body: JSON.stringify({ name: 'status', type: 'enum', enum_id: 'e1' }),
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

// === Enums ===

test('enums.list calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.enums.list()
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums'),
    expect.anything(),
  )
})

test('enums.get calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'e1', name: 'Status' }))

  const result = await api.enums.get('e1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1'),
    expect.anything(),
  )
  expect(result.name).toBe('Status')
})

test('enums.create sends POST with name and values', async () => {
  mockFetch.mockReturnValue(jsonResponse({ id: 'e1', name: 'Status' }))

  await api.enums.create({ name: 'Status', values: ['active', 'inactive'] })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'Status', values: ['active', 'inactive'] }),
    }),
  )
})

test('enums.update sends PUT with name', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'updated' }))

  await api.enums.update('e1', { name: 'New Status' })
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1'),
    expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ name: 'New Status' }),
    }),
  )
})

test('enums.delete sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.enums.delete('e1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

test('enums.listValues calls correct URL', async () => {
  mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

  await api.enums.listValues('e1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1/values'),
    expect.anything(),
  )
})

test('enums.addValue sends POST with value', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'added' }))

  await api.enums.addValue('e1', 'pending')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1/values'),
    expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ value: 'pending' }),
    }),
  )
})

test('enums.removeValue sends DELETE', async () => {
  mockFetch.mockReturnValue(noContentResponse())

  await api.enums.removeValue('e1', 'v1')
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1/values/v1'),
    expect.objectContaining({ method: 'DELETE' }),
  )
})

test('enums.reorderValues sends PUT with ordered_ids', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'reordered' }))

  await api.enums.reorderValues('e1', ['v2', 'v1'])
  expect(mockFetch).toHaveBeenCalledWith(
    expect.stringContaining('/enums/e1/values/reorder'),
    expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ ordered_ids: ['v2', 'v1'] }),
    }),
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
