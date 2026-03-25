import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useCatalogData } from './useCatalogData'

vi.mock('../api/client', () => ({
  api: {
    catalogs: { get: vi.fn() },
    catalogVersions: { listPins: vi.fn() },
    versions: { snapshot: vi.fn() },
    enums: { listValues: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalog = {
  id: 'cat1', name: 'my-catalog', description: 'Test',
  catalog_version_id: 'cv1', catalog_version_label: 'v1.0',
  validation_status: 'draft', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockPins = [
  { entity_type_name: 'model', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 1 },
  { entity_type_name: 'tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
]

const mockSnapshot = {
  entity_type: { id: 'et1', name: 'model' },
  version: { id: 'etv1', version: 1 },
  attributes: [
    { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
    { id: 'a1', name: 'color', type: 'enum', ordinal: 1, required: false, enum_id: 'enum1' },
  ],
  associations: [
    { id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
  ],
}

function TestComponent({ catalogName, changeTab }: { catalogName?: string; changeTab?: string }) {
  const data = useCatalogData(catalogName, 'Admin')
  return (
    <div>
      <span data-testid="loading">{String(data.loading)}</span>
      <span data-testid="error">{data.error || ''}</span>
      <span data-testid="catalog-name">{data.catalog?.name || ''}</span>
      <span data-testid="pins-count">{data.pins.length}</span>
      <span data-testid="active-tab">{data.activeTab}</span>
      <span data-testid="schema-attrs-count">{data.schemaAttrs.length}</span>
      <span data-testid="schema-assocs-count">{data.schemaAssocs.length}</span>
      <span data-testid="enum-keys">{Object.keys(data.enumValues).join(',')}</span>
      {changeTab && <button onClick={() => data.setActiveTab(changeTab)}>Change Tab</button>}
      <button onClick={data.loadCatalog}>Reload</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-19.01: Loads catalog and pins on mount
test('T-19.01: useCatalogData loads catalog and pins on mount', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'red' }, { value: 'blue' }], total: 2 })

  render(<TestComponent catalogName="my-catalog" />)
  await expect.element(page.getByTestId('catalog-name')).toHaveTextContent('my-catalog')
  await expect.element(page.getByTestId('pins-count')).toHaveTextContent('2')
})

// T-19.02: Sets first pin as activeTab when no tab selected
test('T-19.02: useCatalogData sets first pin as activeTab', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [], total: 0 })

  render(<TestComponent catalogName="my-catalog" />)
  await expect.element(page.getByTestId('active-tab')).toHaveTextContent('model')
})

// T-19.03: Loads schema attrs when activeTab changes
test('T-19.03: useCatalogData loads schema when activeTab is set', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [], total: 0 })

  render(<TestComponent catalogName="my-catalog" />)
  await expect.element(page.getByTestId('schema-attrs-count')).toHaveTextContent('2')
  await expect.element(page.getByTestId('schema-assocs-count')).toHaveTextContent('1')
})

// T-19.04: Loads enum values for enum-type attributes
test('T-19.04: useCatalogData loads enum values', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'red' }, { value: 'blue' }], total: 2 })

  render(<TestComponent catalogName="my-catalog" />)
  await expect.element(page.getByTestId('enum-keys')).toHaveTextContent('enum1')
})

// T-19.05: Returns early when name is undefined
test('T-19.05: useCatalogData returns early when name undefined', async () => {
  render(<TestComponent catalogName={undefined} />)
  // Should stop loading quickly with no API calls
  await expect.element(page.getByTestId('loading')).toHaveTextContent('true')
  expect(api.catalogs.get).not.toHaveBeenCalled()
})

// T-19.06: Handles catalog load error
test('T-19.06: useCatalogData handles load error', async () => {
  ;(api.catalogs.get as Mock).mockRejectedValue(new Error('Not found'))

  render(<TestComponent catalogName="bad-catalog" />)
  await expect.element(page.getByTestId('error')).toHaveTextContent('Not found')
  await expect.element(page.getByTestId('loading')).toHaveTextContent('false')
})

// T-19.07: Handles schema load error gracefully
test('T-19.07: useCatalogData handles schema load error', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockRejectedValue(new Error('Schema error'))

  render(<TestComponent catalogName="my-catalog" />)
  // Should not crash, attrs stay empty
  await expect.element(page.getByTestId('schema-attrs-count')).toHaveTextContent('0')
})

// T-19.08: Reloads on loadCatalog call
test('T-19.08: useCatalogData reloads on loadCatalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [], total: 0 })

  render(<TestComponent catalogName="my-catalog" />)
  await expect.element(page.getByTestId('catalog-name')).toHaveTextContent('my-catalog')
  // Initial load + reload
  await page.getByRole('button', { name: 'Reload' }).click()
  // get called at least twice (initial + reload)
  expect((api.catalogs.get as Mock).mock.calls.length).toBeGreaterThanOrEqual(2)
})
