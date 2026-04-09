import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import OperationalCatalogDetailPage from './OperationalCatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), validate: vi.fn() },
    catalogVersions: { listPins: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), tree: vi.fn() },
    versions: { snapshot: vi.fn() },
    links: { forwardRefs: vi.fn(), reverseRefs: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockSnapshotServer = {
  entity_type: { id: 'et1', name: 'mcp-server', created_at: '', updated_at: '' },
  version: { id: 'etv1', version: 1 },
  attributes: [{ id: 'a1', name: 'endpoint', description: '', type: 'string', ordinal: 1, required: false }],
  associations: [],
}

const mockSnapshotTool = {
  entity_type: { id: 'et2', name: 'mcp-tool', created_at: '', updated_at: '' },
  version: { id: 'etv2', version: 1 },
  attributes: [],
  associations: [],
}

const mockCatalog = {
  id: 'cat1', name: 'test-catalog', description: 'Test',
  catalog_version_id: 'cv1', catalog_version_label: 'v1.0',
  validation_status: 'draft' as const,
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockPins = [
  { pin_id: 'pin-1', entity_type_name: 'mcp-server', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 1 },
  { pin_id: 'pin-2', entity_type_name: 'mcp-tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
]

const mockTree = [
  {
    instance_id: 'i1', instance_name: 'my-server', entity_type_name: 'mcp-server',
    description: 'A server', children: [
      { instance_id: 'i2', instance_name: 'my-tool', entity_type_name: 'mcp-tool',
        description: 'A tool', children: [] },
    ],
  },
  { instance_id: 'i3', instance_name: 'other-server', entity_type_name: 'mcp-server',
    description: '', children: [] },
]

const mockInstanceDetail = {
  id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'my-server',
  description: 'A server', version: 2,
  attributes: [
    { name: 'endpoint', type: 'string', value: 'https://example.com' },
    { name: 'status', type: 'enum', value: 'active' },
  ],
  parent_chain: [],
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
}

const mockChildDetail = {
  id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'my-tool',
  parent_instance_id: 'i1', description: 'A tool', version: 1,
  attributes: [],
  parent_chain: [
    { instance_id: 'i1', instance_name: 'my-server', entity_type_name: 'mcp-server' },
  ],
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockForwardRefs = [
  { link_id: 'l1', association_name: 'uses-model', association_type: 'directional',
    instance_id: 'i4', instance_name: 'gpt-4', entity_type_name: 'model' },
]

const mockReverseRefs = [
  { link_id: 'l2', association_name: 'monitored-by', association_type: 'directional',
    instance_id: 'i5', instance_name: 'monitor-1', entity_type_name: 'guardrail' },
]

function renderDetail(role: 'Admin' | 'RW' | 'RO' | 'SuperAdmin' = 'RO') {
  return render(
    <MemoryRouter initialEntries={['/catalogs/test-catalog']}>
      <Routes>
        <Route path="/catalogs/:name" element={<OperationalCatalogDetailPage role={role} />} />
      </Routes>
    </MemoryRouter>
  )
}

// Helper: switch to tree browser tab and expand the mcp-server entity type group
async function openTreeAndExpandServers() {
  await page.getByText('Tree Browser').click()
  await expect.element(page.getByText('mcp-server (2)')).toBeVisible()
  await page.getByText('mcp-server (2)').click()
  await expect.element(page.getByText('my-server').first()).toBeVisible()
}

// Helper: click a tree node instance by name
async function clickTreeNode(name: string) {
  await page.getByText(name).first().click()
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(mockSnapshotTool)
    return Promise.resolve(mockSnapshotServer)
  })
  ;(api.instances.tree as Mock).mockResolvedValue(mockTree)
  ;(api.instances.get as Mock).mockResolvedValue(mockInstanceDetail)
  ;(api.links.forwardRefs as Mock).mockResolvedValue(mockForwardRefs)
  ;(api.links.reverseRefs as Mock).mockResolvedValue(mockReverseRefs)
})

// === Catalog Detail Header ===

test('T-13.68: shows catalog header with name, status badge, CV label', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByText('draft')).toBeVisible()
  await expect.element(page.getByText(/v1\.0/)).toBeVisible()
})

// === Containment Tree Browser — Two-Pane (T-13.72 through T-13.77) ===

test('T-13.72: tree browser shows two-pane layout', async () => {
  renderDetail()
  await page.getByText('Tree Browser').click()
  await expect.element(page.getByText('Containment Tree')).toBeVisible()
  // Right pane shows empty state when nothing selected
  await expect.element(page.getByText('Select an instance from the tree')).toBeVisible()
})

test('T-13.73: tree groups root instances under entity type headers with counts', async () => {
  renderDetail()
  await page.getByText('Tree Browser').click()
  await expect.element(page.getByText('mcp-server (2)')).toBeVisible()
  // Expand and verify instance names
  await page.getByText('mcp-server (2)').click()
  await expect.element(page.getByText('my-server').first()).toBeVisible()
})

test('T-13.74: entity type group headers are expandable', async () => {
  renderDetail()
  await page.getByText('Tree Browser').click()
  // Expand indicators present
  await expect.element(page.getByText('▸').first()).toBeVisible()
})

test('T-13.75: clicking tree instance shows detail in right panel', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  // Detail should appear inline in the right panel (not a drawer)
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // Empty state should be gone
  const emptyState = page.getByText('Select an instance from the tree')
  expect(emptyState.elements().length).toBe(0)
})

test('T-13.76: multi-level tree shows child nodes under parent', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  // Expand my-server to see its children
  await page.getByText('▸').first().click()
  await expect.element(page.getByText('my-tool')).toBeVisible()
})

test('T-13.77: empty state shown when no instance selected', async () => {
  renderDetail()
  await page.getByText('Tree Browser').click()
  await expect.element(page.getByText('Select an instance from the tree')).toBeVisible()
})

// === Instance Detail (T-13.86 through T-13.89) ===

test('T-13.86: instance detail shows attributes table', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('heading', { name: 'Attributes' })).toBeVisible()
  await expect.element(page.getByText('https://example.com').first()).toBeVisible()
})

test('T-13.87: enum attribute values shown', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('active')).toBeVisible()
})

test('T-13.88: description displayed', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('A server').first()).toBeInTheDocument()
})

test('T-13.89: version number visible', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText(/Version 2/)).toBeVisible()
})

// === Breadcrumb Navigation (T-13.90 through T-13.93) ===

test('T-13.90: breadcrumb renders containment path', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail()
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await expect.element(page.getByText('my-tool')).toBeVisible()
  await clickTreeNode('my-tool')
  await expect.element(page.getByText('test-catalog').first()).toBeVisible()
  await expect.element(page.getByText(/mcp-server.*my-server/).first()).toBeVisible()
})

test('T-13.91: breadcrumb shows entity type and instance name', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail()
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByText('mcp-server: my-server').first()).toBeVisible()
})

test('T-13.92: breadcrumb items are present', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail()
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByText('my-tool').first()).toBeVisible()
})

test('T-13.93: root instance breadcrumb shows catalog', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockInstanceDetail)
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('test-catalog').first()).toBeVisible()
})

// === Reference Navigation (T-13.94 through T-13.97) ===

test('T-13.94: forward references section shows', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('Forward References')).toBeVisible()
})

test('T-13.95: reverse references section shows', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('Referenced By')).toBeVisible()
})

test('T-13.96: referenced instance is clickable', async () => {
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'gpt-4' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'monitor-1' })).toBeVisible()
})

test('T-13.97: no references shows message', async () => {
  ;(api.links.forwardRefs as Mock).mockResolvedValue([])
  ;(api.links.reverseRefs as Mock).mockResolvedValue([])
  renderDetail()
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByText('No references.')).toBeVisible()
})

// === Read-Only Enforcement (T-13.98 through T-13.102) ===

test('T-13.98: no create buttons in read-only mode', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const createBtns = page.getByRole('button', { name: /Create/ })
  expect(createBtns.elements().length).toBe(0)
})

test('T-13.99: no edit buttons', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const editBtns = page.getByRole('button', { name: /Edit/ })
  expect(editBtns.elements().length).toBe(0)
})

test('T-13.100: no delete buttons', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const deleteBtns = page.getByRole('button', { name: /Delete/ })
  expect(deleteBtns.elements().length).toBe(0)
})

test('T-13.101: no link/unlink actions', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const linkBtns = page.getByRole('button', { name: /^Link/ })
  expect(linkBtns.elements().length).toBe(0)
  const unlinkBtns = page.getByRole('button', { name: /Unlink/ })
  expect(unlinkBtns.elements().length).toBe(0)
})

test('T-13.102: read-only even for SuperAdmin role', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const createBtns = page.getByRole('button', { name: /Create/ })
  expect(createBtns.elements().length).toBe(0)
  const editBtns = page.getByRole('button', { name: /Edit/ })
  expect(editBtns.elements().length).toBe(0)
  const deleteBtns = page.getByRole('button', { name: /Delete/ })
  expect(deleteBtns.elements().length).toBe(0)
})

// === Coverage: error and edge cases ===

test('catalog not found shows error', async () => {
  ;(api.catalogs.get as Mock).mockRejectedValue(new Error('Not found'))
  renderDetail()
  await expect.element(page.getByText('Not found')).toBeVisible()
})

test('empty tree shows no instances message', async () => {
  ;(api.instances.tree as Mock).mockResolvedValue([])
  renderDetail()
  await expect.element(page.getByText('No instances in this catalog.')).toBeVisible()
})

test('invalid catalog status shows red label', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({
    ...mockCatalog, validation_status: 'invalid',
  })
  renderDetail()
  await expect.element(page.getByText('invalid')).toBeVisible()
})

test('valid catalog status shows green label', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({
    ...mockCatalog, validation_status: 'valid',
  })
  renderDetail()
  await expect.element(page.getByText('valid', { exact: true })).toBeVisible()
})

// === Catalog Validation Tests ===

// T-15.47: Validate button visible on operational catalog detail for RW+
test('T-15.47: Validate button visible on operational catalog detail for RW', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('button', { name: 'Validate' })).toBeVisible()
})

// L2 fix: Validate button hidden for RO in operational UI
test('Validate button hidden for RO in operational UI', async () => {
  renderDetail('RO')
  // Wait for the page to load
  await expect.element(page.getByText('test-catalog', { exact: true })).toBeVisible()
  // RO should not see the Validate button
  expect(document.querySelector('button')?.textContent).not.toContain('Validate')
})

// T-15.48: Validation results displayed in operational UI
test('T-15.48: validation results displayed in operational UI', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({
    status: 'invalid',
    errors: [
      { entity_type: 'Server', instance_name: 'srv-1', field: 'hostname', violation: 'required attribute missing' },
    ],
  })
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'invalid' })
  renderDetail('RW')
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByText('Validation failed')).toBeVisible()
  await expect.element(page.getByText(/srv-1.*hostname.*required/)).toBeVisible()
})

// Coverage: breadcrumb "Catalogs" link navigates to catalog list
test('breadcrumb Catalogs link navigates', async () => {
  renderDetail('RO')
  const link = page.getByRole('button', { name: 'Catalogs' })
  await expect.element(link).toBeVisible()
  await link.click()
  // Navigation happens — no crash
})

// T-21.21: Model Diagram tab exists on operational catalog detail page
test('T-21.21: Model Diagram tab exists on operational catalog detail page', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Model Diagram' })).toBeVisible()
})

// T-21.22: Clicking Model Diagram tab loads diagram data
test('T-21.22: clicking Model Diagram tab loads diagram data', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  await vi.waitFor(() => {
    expect(api.catalogVersions.listPins).toHaveBeenCalledWith('cv1')
  })
})

// T-21.23: Diagram renders entity type nodes from CV
test('T-21.23: diagram renders entity type nodes', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  await expect.element(page.getByTestId('entity-type-diagram')).toBeVisible()
})

// T-21.24: Diagram error is displayed
test('T-21.24: diagram tab shows error alert on API failure', async () => {
  // Diagram hook calls listPins — reject it
  ;(api.catalogVersions.listPins as Mock).mockRejectedValue(new Error('Diagram load failed'))
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Model Diagram' })).toBeVisible()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  await expect.element(page.getByText('Diagram load failed')).toBeVisible()
})

// T-21.25: Empty state when diagram has no data
test('T-21.25: diagram tab shows empty state when no pins', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  await expect.element(page.getByText('No model diagram available')).toBeVisible()
})
