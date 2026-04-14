import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useEntityTypeData } from './useEntityTypeData'

vi.mock('../api/client', () => ({
  api: {
    entityTypes: { get: vi.fn(), list: vi.fn() },
    attributes: { list: vi.fn() },
    associations: { list: vi.fn() },
    versions: { list: vi.fn(), diff: vi.fn() },
    typeDefinitions: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockEntityType = {
  id: 'et-1', name: 'MLModel',
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
}

const mockAttributes = [
  { id: 'a1', name: 'hostname', description: 'The host', type: 'string', ordinal: 0, required: false },
]

const mockAssociations = [
  { id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et-2' },
]

const mockVersions = [
  { id: 'v1', entity_type_id: 'et-1', version: 1, description: 'Initial', created_at: '2026-01-01T00:00:00Z' },
]

const mockTypeDefinitions = [
  { id: 'td1', name: 'Colors', base_type: 'enum', system: false, latest_version: 1, latest_version_id: 'tdv-auto', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

const mockEntityTypes = [
  { id: 'et-1', name: 'MLModel' },
  { id: 'et-2', name: 'Tool' },
]

function TestComponent({ entityTypeId, initialTab }: { entityTypeId?: string; initialTab?: string }) {
  const data = useEntityTypeData(entityTypeId, initialTab)
  return (
    <div>
      <span data-testid="loading">{String(data.loading)}</span>
      <span data-testid="error">{data.error || ''}</span>
      <span data-testid="et-name">{data.entityType?.name || ''}</span>
      <span data-testid="attrs-count">{data.attributes.length}</span>
      <span data-testid="attrs-loading">{String(data.attrsLoading)}</span>
      <span data-testid="assocs-count">{data.associations.length}</span>
      <span data-testid="assocs-loading">{String(data.assocsLoading)}</span>
      <span data-testid="versions-count">{data.versions.length}</span>
      <span data-testid="versions-loading">{String(data.versionsLoading)}</span>
      <span data-testid="td-count">{data.typeDefinitions.length}</span>
      <span data-testid="entity-types-count">{data.entityTypes.length}</span>
      <span data-testid="active-tab">{String(data.activeTab)}</span>
      <button onClick={() => data.setActiveTab('attributes')}>Go Attributes</button>
      <button onClick={() => data.setActiveTab('associations')}>Go Associations</button>
      <button onClick={() => data.setActiveTab('versions')}>Go Versions</button>
      <button onClick={data.loadEntityType}>Reload ET</button>
      <button onClick={data.loadAttributes}>Reload Attrs</button>
      <button onClick={data.loadAssociations}>Reload Assocs</button>
      <button onClick={data.loadVersions}>Reload Versions</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.01: Loads entity type on mount
test('T-20.01: useEntityTypeData loads entity type on mount', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('MLModel')
  await expect.element(page.getByTestId('loading')).toHaveTextContent('false')
})

// T-20.02: Shows error when entity type fails to load
test('T-20.02: useEntityTypeData shows error on load failure', async () => {
  ;(api.entityTypes.get as Mock).mockRejectedValue(new Error('Not found'))

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('error')).toHaveTextContent('Not found')
})

// T-20.03: Loads attributes when switching to attributes tab
test('T-20.03: useEntityTypeData loads attributes on tab switch', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)
  ;(api.attributes.list as Mock).mockResolvedValue({ items: mockAttributes, total: 1 })
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: mockTypeDefinitions, total: 1 })

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('MLModel')

  await page.getByRole('button', { name: 'Go Attributes' }).click()
  await expect.element(page.getByTestId('attrs-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('td-count')).toHaveTextContent('1')
})

// T-20.04: Loads associations when switching to associations tab
test('T-20.04: useEntityTypeData loads associations on tab switch', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)
  ;(api.associations.list as Mock).mockResolvedValue({ items: mockAssociations, total: 1 })
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: mockEntityTypes, total: 2 })

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('MLModel')

  await page.getByRole('button', { name: 'Go Associations' }).click()
  await expect.element(page.getByTestId('assocs-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('entity-types-count')).toHaveTextContent('2')
})

// T-20.05: Loads versions when switching to versions tab
test('T-20.05: useEntityTypeData loads versions on tab switch', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 1 })

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('MLModel')

  await page.getByRole('button', { name: 'Go Versions' }).click()
  await expect.element(page.getByTestId('versions-count')).toHaveTextContent('1')
})

// T-20.06: Respects initial tab
test('T-20.06: useEntityTypeData respects initial tab', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)
  ;(api.attributes.list as Mock).mockResolvedValue({ items: mockAttributes, total: 1 })
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: mockTypeDefinitions, total: 1 })

  render(<TestComponent entityTypeId="et-1" initialTab="attributes" />)
  await expect.element(page.getByTestId('active-tab')).toHaveTextContent('attributes')
  await expect.element(page.getByTestId('attrs-count')).toHaveTextContent('1')
})

// T-20.07: Does nothing when entityTypeId is undefined
test('T-20.07: useEntityTypeData does nothing without id', async () => {
  render(<TestComponent />)
  await expect.element(page.getByTestId('loading')).toHaveTextContent('true')
  expect(api.entityTypes.get).not.toHaveBeenCalled()
})

// T-20.08: Reload entity type
test('T-20.08: useEntityTypeData reload works', async () => {
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)

  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('MLModel')

  ;(api.entityTypes.get as Mock).mockResolvedValue({ ...mockEntityType, name: 'Updated' })
  await page.getByRole('button', { name: 'Reload ET' }).click()
  await expect.element(page.getByTestId('et-name')).toHaveTextContent('Updated')
})
