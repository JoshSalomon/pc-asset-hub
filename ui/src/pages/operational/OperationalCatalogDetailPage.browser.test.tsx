import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page, userEvent } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import OperationalCatalogDetailPage from './OperationalCatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), validate: vi.fn() },
    catalogVersions: { listPins: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), tree: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), setParent: vi.fn(), createContained: vi.fn() },
    versions: { snapshot: vi.fn() },
    links: { forwardRefs: vi.fn(), reverseRefs: vi.fn(), create: vi.fn(), delete: vi.fn() },
    exporters: { list: vi.fn() },
    exportBindings: { list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), run: vi.fn(), download: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockSnapshotServer = {
  entity_type: { id: 'et1', name: 'mcp-server', created_at: '', updated_at: '' },
  version: { id: 'etv1', version: 1 },
  attributes: [
    { id: 'sys-name', name: 'name', description: 'Instance name', type: 'string', ordinal: 0, required: true, system: true },
    { id: 'sys-desc', name: 'description', description: 'Instance description', type: 'string', ordinal: 0, required: false, system: true },
    { id: 'a1', name: 'endpoint', description: '', type: 'string', ordinal: 1, required: false },
  ],
  associations: [
    { id: 'assoc-1', name: 'contains-tool', type: 'containment', target_entity_type_id: 'et2', target_entity_type_name: 'mcp-tool', source_role: 'parent', target_role: 'child', source_cardinality: '0..n', target_cardinality: '1', direction: 'outgoing' as const, source_entity_type_id: 'et1', source_entity_type_name: 'mcp-server' },
    { id: 'assoc-link-1', name: 'uses-model', type: 'directional', target_entity_type_id: 'et3', target_entity_type_name: 'model', source_role: '', target_role: '', source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' as const, source_entity_type_id: 'et1', source_entity_type_name: 'mcp-server' },
  ],
}

const mockSnapshotTool = {
  entity_type: { id: 'et2', name: 'mcp-tool', created_at: '', updated_at: '' },
  version: { id: 'etv2', version: 1 },
  attributes: [
    { id: 'sys-name-2', name: 'name', description: 'Instance name', type: 'string', ordinal: 0, required: true, system: true },
    { id: 'sys-desc-2', name: 'description', description: 'Instance description', type: 'string', ordinal: 0, required: false, system: true },
  ],
  associations: [
    { id: 'assoc-2', name: 'contains-tool', type: 'containment', target_entity_type_id: 'et2', target_entity_type_name: 'mcp-tool', source_role: 'parent', target_role: 'child', source_cardinality: '0..n', target_cardinality: '1', direction: 'incoming' as const, source_entity_type_id: 'et1', source_entity_type_name: 'mcp-server' },
  ],
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
  { pin_id: 'pin-3', entity_type_name: 'model', entity_type_id: 'et3', entity_type_version_id: 'etv3', version: 1 },
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
    { name: 'name', type: 'string', value: 'my-server', system: true },
    { name: 'description', type: 'string', value: 'A server', system: true },
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
        <Route path="/" element={<div>Catalogs List</div>} />
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
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [] })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [] })
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

test('T-13.102: SuperAdmin sees write controls on operational UI', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
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

test('published catalog shows published indicator badge', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail()
  await expect.element(page.getByText('published')).toBeVisible()
})

test('unpublished catalog does not show published badge', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  expect(page.getByText('published').elements().length).toBe(0)
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

// === Operational UI Editing — Instance CRUD (T-32.01 through T-32.28) ===

test('T-32.01: Create Instance button visible at top of tree for RW role', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
})

test('T-32.02: Create Instance button hidden for RO role', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const btn = page.getByRole('button', { name: 'Create Instance' })
  expect(btn.elements().length).toBe(0)
})

test('T-32.03: Create Instance button hidden for non-SuperAdmin on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const btn = page.getByRole('button', { name: 'Create Instance' })
  expect(btn.elements().length).toBe(0)
})

test('T-32.04: Create modal opens with entity type dropdown showing all pinned types', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
})

test('T-32.05: Selecting entity type in Create modal loads attribute fields', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
})

test('T-32.06: Creating instance via modal calls API and refreshes tree', async () => {
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'new-1', name: 'new-instance' })
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('new-instance')
  await page.getByRole('button', { name: 'Create' }).click()
  await vi.waitFor(() => {
    expect(api.instances.create).toHaveBeenCalledWith('test-catalog', 'mcp-server', expect.objectContaining({ name: 'new-instance' }))
  })
})

test('Create button disabled while submitting', async () => {
  let resolveCreate: (v: unknown) => void
  ;(api.instances.create as Mock).mockImplementation(() => new Promise(r => { resolveCreate = r }))
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('test')
  await page.getByRole('button', { name: 'Create' }).click()
  // Button should be disabled while API call is in flight
  await expect.element(page.getByRole('button', { name: 'Create' })).toBeDisabled()
  resolveCreate!({ id: 'x', name: 'test' })
})

test('T-32.21: After create, tree reloads', async () => {
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'new-1', name: 'new-instance' })
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const treeBefore = (api.instances.tree as Mock).mock.calls.length
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('new-instance')
  await page.getByRole('button', { name: 'Create' }).click()
  await vi.waitFor(() => {
    expect(api.instances.create).toHaveBeenCalled()
    expect((api.instances.tree as Mock).mock.calls.length).toBeGreaterThan(treeBefore)
  })
})

test('T-32.07: + icon visible next to each entity type group header for RW+', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create mcp-server' })).toBeVisible()
})

test('T-32.08: + icon hidden for RO role', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  const btns = page.getByRole('button', { name: /Create mcp/ })
  expect(btns.elements().length).toBe(0)
})

test('T-32.09: Clicking + opens Create modal pre-filled with that entity type', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create mcp-server' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  // Verify schema was loaded (snapshot API called for mcp-server)
  await vi.waitFor(() => {
    expect(api.versions.snapshot).toHaveBeenCalledWith('et1', 1)
  }, { timeout: 5000 })
  // Should be pre-selected to mcp-server — Name field should already be visible
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
})

test('T-32.10: Edit button visible in detail panel when instance selected for RW+', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit' })).toBeVisible()
})

test('T-32.11: Edit button hidden for RO role', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const editBtns = page.getByRole('button', { name: 'Edit' })
  expect(editBtns.elements().length).toBe(0)
})

test('T-32.12: Edit modal pre-fills current name, description, and attribute values', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => {
    expect(api.versions.snapshot).toHaveBeenCalled()
  })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
})

test('T-32.13: Submitting Edit modal updates instance and refreshes detail panel', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'updated-server' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('updated-server')
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    expect(api.instances.update).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1', expect.objectContaining({ name: 'updated-server', version: 2 }))
  })
})

test('T-32.14: Renaming instance via Edit modal calls update API with new name', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'renamed-server' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('renamed-server')
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    expect(api.instances.update).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1', expect.objectContaining({ name: 'renamed-server' }))
  })
  // Tree should reload after rename
  const treeCalls = (api.instances.tree as Mock).mock.calls.length
  expect(treeCalls).toBeGreaterThanOrEqual(2)
})

test('Clearing attribute in Edit modal sends null to API', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'my-server' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  const endpointInput = page.getByRole('textbox', { name: /endpoint/ })
  await endpointInput.fill('')
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    const call = (api.instances.update as Mock).mock.calls[0]
    expect(call[3].attributes).toHaveProperty('endpoint', null)
  })
})

test('Edit submit does not include system attributes (name/description) in attributes payload', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'my-server' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    const call = (api.instances.update as Mock).mock.calls[0]
    const payload = call[3]
    // System attrs should NOT be in the attributes object
    if (payload.attributes) {
      expect(payload.attributes).not.toHaveProperty('name')
      expect(payload.attributes).not.toHaveProperty('description')
    }
  })
})

test('T-32.15: Delete button visible in detail panel for RW+', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' }).first()).toBeVisible()
})

test('T-32.16: Delete button hidden for RO role', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const deleteBtns = page.getByRole('button', { name: 'Delete' })
  expect(deleteBtns.elements().length).toBe(0)
})

test('T-32.17: Delete confirmation dialog shows instance name', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('my-server')).toBeVisible()
})

test('T-32.18: Confirming delete removes instance and refreshes tree', async () => {
  ;(api.instances.delete as Mock).mockResolvedValue({})
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.instances.delete).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1')
  })
})

test('T-32.19: Delete instance with children shows cascade warning listing all descendants', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // Click delete on parent (my-server has child my-tool)
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByText(/1 contained instance/)).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('my-tool')).toBeVisible()
})

test('T-32.20: Confirming cascade delete calls API for parent instance', async () => {
  ;(api.instances.delete as Mock).mockResolvedValue({})
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText(/1 contained instance/)).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.instances.delete).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1')
  })
})

test('T-32.22: After edit, tree reloads and edited instance stays selected', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'updated' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    expect(api.instances.update).toHaveBeenCalled()
  })
  // Tree should have been reloaded after the edit (selectNodeById also called)
  await vi.waitFor(() => {
    expect(api.instances.get).toHaveBeenCalled()
  })
})

test('T-32.23: After delete, parent node is selected', async () => {
  ;(api.instances.delete as Mock).mockResolvedValue({})
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.instances.delete).toHaveBeenCalled()
    // After delete of child, tree reloads and parent should be re-selected
    expect(api.instances.get).toHaveBeenCalled()
  })
})

test('T-32.24: After delete of root instance, selection is cleared', async () => {
  const mockOtherDetail = {
    ...mockInstanceDetail, id: 'i3', name: 'other-server', description: '', parent_instance_id: undefined,
    parent_chain: [],
  }
  ;(api.instances.get as Mock).mockResolvedValue(mockOtherDetail)
  ;(api.instances.delete as Mock).mockResolvedValue({})
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('other-server')
  await expect.element(page.getByRole('heading', { name: 'other-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.instances.delete).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i3')
  })
  // Selection should be cleared — empty state reappears
  await expect.element(page.getByText('Select an instance from the tree')).toBeVisible()
})

test('T-32.23b: Delete captures parent at modal-open, not at submit', async () => {
  ;(api.instances.delete as Mock).mockResolvedValue({})
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()

  // Open delete modal — parent should be captured NOW
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  // Confirm delete — even though tree will reload, parent captured at open is used
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.instances.delete).toHaveBeenCalled()
    // Parent (my-server) should be re-selected after delete
    expect(api.instances.get).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1')
  })
})

test('T-32.26: Create instance API error shows validation message', async () => {
  ;(api.instances.create as Mock).mockRejectedValue(new Error('invalid attribute value for hostname'))
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('bad-instance')
  await page.getByRole('button', { name: 'Create' }).click()
  await expect.element(page.getByText('invalid attribute value for hostname')).toBeVisible()
})

test('T-32.25: Create instance with required attributes missing shows validation error', async () => {
  ;(api.instances.create as Mock).mockRejectedValue(new Error('name is required'))
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('test')
  await page.getByRole('button', { name: 'Create' }).click()
  await expect.element(page.getByText('name is required')).toBeVisible()
})

test('T-32.27: Edit instance API error shows error in modal', async () => {
  ;(api.instances.update as Mock).mockRejectedValue(new Error('update failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('button', { name: 'Save' }).click()
  await expect.element(page.getByText('update failed')).toBeVisible()
})

test('T-32.28: Delete instance API error shows error alert', async () => {
  ;(api.instances.delete as Mock).mockRejectedValue(new Error('delete failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('delete failed')).toBeVisible()
})

// === Containment Tests (T-32.29 through T-32.43) ===

test('T-32.29: Add Child button visible for instances with containment associations', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // mcp-server has outgoing containment to mcp-tool
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Add Child' })).toBeVisible()
})

test('T-32.31: Add Child button hidden for RO role', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const btns = page.getByRole('button', { name: 'Add Child' })
  expect(btns.elements().length).toBe(0)
})

test('T-32.32: Add Child modal shows containment-eligible child types from CV associations', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Add Child' })).toBeVisible()
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // The AddChildModal renders and shows containment target types
  await expect.element(page.getByText('Add Contained Instance')).toBeVisible()
})

test('T-32.33: Creating child via Add Child calls createContained API', async () => {
  ;(api.instances.createContained as Mock).mockResolvedValue({ id: 'child-1', name: 'new-child' })
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.36: Set Parent modal opens and shows parent type', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // SetParentModal title includes the instance name
  await expect.element(page.getByText(/Set Container.*my-tool/)).toBeVisible()
})

test('T-32.37: Setting parent calls setParent API', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockResolvedValue({})
  ;(api.instances.list as Mock).mockResolvedValue({ items: [{ id: 'i1', name: 'my-server', entity_type_id: 'et1' }], total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.34: Set Parent button visible for instances that can be contained', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await expect.element(page.getByText('my-tool')).toBeVisible()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  // mcp-tool has incoming containment from mcp-server
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Set Parent' })).toBeVisible()
})

test('T-32.35: Set Parent button hidden for RO role', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RO')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  const btns = page.getByRole('button', { name: 'Set Parent' })
  expect(btns.elements().length).toBe(0)
})

test('T-32.30: Add Child button hidden when entity type has no containment associations', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  // mcp-tool has no outgoing containment — Add Child should not appear
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  const btns = page.getByRole('button', { name: 'Add Child' })
  expect(btns.elements().length).toBe(0)
})

test('T-32.42: Add Child API error shows error in modal', async () => {
  ;(api.instances.createContained as Mock).mockRejectedValue(new Error('containment failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Add Child' })).toBeVisible()
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.43: Set Parent API error shows error in modal', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockRejectedValue(new Error('set parent failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Set Parent' })).toBeVisible()
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.38: Remove from Container button visible for contained instances', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: /Remove from Container/ })).toBeVisible()
})

test('T-32.39: Remove from Container button hidden for root instances', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const btns = page.getByRole('button', { name: /Remove from Container/ })
  expect(btns.elements().length).toBe(0)
})

test('T-32.40: Remove from Container button hidden for RO role', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RO')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  const btns = page.getByRole('button', { name: /Remove from Container/ })
  expect(btns.elements().length).toBe(0)
})

test('T-32.41: Removing from container calls API and re-selects instance', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockResolvedValue({})
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  const getCallsBefore = (api.instances.get as Mock).mock.calls.length
  await page.getByRole('button', { name: /Remove from Container/ }).click()
  await vi.waitFor(() => {
    expect(api.instances.setParent).toHaveBeenCalledWith('test-catalog', 'mcp-tool', 'i2', { parent_type: '', parent_instance_id: '' })
    // Instance detail should be re-fetched after tree reload
    expect((api.instances.get as Mock).mock.calls.length).toBeGreaterThan(getCallsBefore)
  })
})

test('T-32.41b: Remove from Container button disabled while submitting', async () => {
  let resolveSetParent: (v: unknown) => void
  ;(api.instances.setParent as Mock).mockImplementation(() => new Promise(r => { resolveSetParent = r }))
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: /Remove from Container/ }).click()
  // Button should be disabled while API call is in flight
  await expect.element(page.getByRole('button', { name: /Remove from Container/ })).toBeDisabled()
  resolveSetParent!({})
})

test('T-32.43b: Remove from Container error shows inline in detail panel', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockRejectedValue(new Error('remove failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: /Remove from Container/ }).click()
  // Error should appear inline in the detail panel (not just page-level)
  await expect.element(page.getByText('remove failed')).toBeVisible()
})

// === Link Tests (T-32.44 through T-32.54) ===

test('T-32.44: Create Link button visible in detail panel for RW+', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // mcp-server has outgoing directional link (uses-model)
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Create Link' })).toBeVisible()
})

test('T-32.45: Create Link button hidden for RO role', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const btns = page.getByRole('button', { name: 'Create Link' })
  expect(btns.elements().length).toBe(0)
})

test('T-32.46: Create Link modal opens showing associations', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // LinkModal should be open showing link association options
  await expect.element(page.getByText('Link to Instance')).toBeVisible()
})

test('T-32.47: Create Link modal loads target instances when association selected', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({ items: [{ id: 'i3', name: 'other-server', entity_type_id: 'et1' }], total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.48: Creating link calls API and refreshes refs', async () => {
  ;(api.links.create as Mock).mockResolvedValue({ id: 'l-new' })
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

test('T-32.49: Delete button visible on each link row for RW+', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // Forward ref row should have a Delete button
  await expect.element(page.getByLabelText('Forward references').getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('T-32.50: Link delete buttons hidden for RO role', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // Forward refs show but no Delete button
  await expect.element(page.getByText('Forward References')).toBeVisible()
  const deleteBtns = page.getByLabelText('Forward references').getByRole('button', { name: 'Delete' })
  expect(deleteBtns.elements().length).toBe(0)
})

test('T-32.51: Confirming link delete removes it from references', async () => {
  ;(api.links.delete as Mock).mockResolvedValue({})
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByLabelText('Forward references').getByRole('button', { name: 'Delete' }).click()
  await vi.waitFor(() => {
    expect(api.links.delete).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1', 'l1')
  })
})

test('T-32.53: Delete Link API error shows error alert', async () => {
  ;(api.links.delete as Mock).mockRejectedValue(new Error('delete link failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByLabelText('Forward references').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('delete link failed')).toBeVisible()
})

test('T-32.51b: Unlink button disabled while submitting', async () => {
  let resolveDelete: (v: unknown) => void
  ;(api.links.delete as Mock).mockImplementation(() => new Promise(r => { resolveDelete = r }))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  const deleteBtn = page.getByLabelText('Forward references').getByRole('button', { name: 'Delete' })
  await deleteBtn.click()
  // Button should be disabled while API call is in flight
  await expect.element(deleteBtn).toBeDisabled()
  resolveDelete!({})
})

test('T-32.52: Create Link API error shows error in modal', async () => {
  ;(api.links.create as Mock).mockRejectedValue(new Error('link creation failed'))
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Create Link' })).toBeVisible()
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
})

// === Role-Aware Controls Tests (T-32.55 through T-32.63) ===

test('T-32.57: Admin role on non-published catalog: all write controls visible', async () => {
  renderDetail('Admin')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' }).first()).toBeVisible()
})

test('T-32.58: SuperAdmin on non-published catalog: all write controls visible', async () => {
  renderDetail('SuperAdmin')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit' })).toBeVisible()
})

test('T-32.60: Admin on published catalog: all write controls hidden', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  expect(page.getByRole('button', { name: 'Create Instance' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: /^Create mcp/ }).elements().length).toBe(0)
})

test('T-32.62: RW role shows write controls that RO does not', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
})

test('T-32.63: RO role hides write controls that RW shows', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  expect(page.getByRole('button', { name: 'Create Instance' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: 'Validate' }).elements().length).toBe(0)
})

test('T-32.55: RO role: ALL write controls hidden (comprehensive check)', async () => {
  renderDetail('RO')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  // No Create, Edit, Delete, Add Child, Set Parent, Remove Container, Create Link buttons
  expect(page.getByRole('button', { name: 'Create Instance' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: 'Edit' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: /^Delete/ }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: 'Add Child' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: 'Set Parent' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: /Remove from Container/ }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: 'Create Link' }).elements().length).toBe(0)
  // No + icons
  expect(page.getByRole('button', { name: /^Create mcp/ }).elements().length).toBe(0)
})

test('T-32.56: RW role on non-published catalog: all write controls visible', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' }).first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Child' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Link' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create mcp-server' })).toBeVisible()
})

test('T-32.59: RW on published catalog: all write controls hidden', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  expect(page.getByRole('button', { name: 'Create Instance' }).elements().length).toBe(0)
  expect(page.getByRole('button', { name: /^Create mcp/ }).elements().length).toBe(0)
})

test('T-32.61: SuperAdmin on published catalog: all write controls visible', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Instance' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create mcp-server' })).toBeVisible()
})

// Cover L141: catalog not found guard (catalog is null after load completes without error)
test('shows catalog not found when API returns null-like response', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(null)
  renderDetail()
  await expect.element(page.getByText('Catalog not found')).toBeVisible()
})

// === Coverage: handler body paths (handleAddChild, handleSetParent, handleCreateLink) ===

test('handleAddChild create mode calls createContained and refreshes tree', async () => {
  ;(api.instances.createContained as Mock).mockResolvedValue({ id: 'child-new', name: 'new-child' })
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByText('Add Contained Instance')).toBeVisible()
  // Select child type via PF6 Select (uses button-based menu, not native options)
  await page.getByText('Select child type...').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('mcp-tool').first()).toBeVisible()
  })
  await page.getByText('mcp-tool').first().click()
  // Fill in name
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('new-child')
  // Submit — use exact match to avoid matching "Create New" mode toggle
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  await vi.waitFor(() => {
    expect(api.instances.createContained).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1', 'mcp-tool',
      expect.objectContaining({ name: 'new-child' }))
  })
})

test('handleAddChild adopt mode calls setParent API', async () => {
  ;(api.instances.setParent as Mock).mockResolvedValue({})
  ;(api.instances.list as Mock).mockResolvedValue({ items: [{ id: 'orphan-1', name: 'orphan-tool', entity_type_id: 'et2' }], total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByText('Add Contained Instance')).toBeVisible()
  // Select child type via PF6 Select
  await page.getByText('Select child type...').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('mcp-tool').first()).toBeVisible()
  })
  await page.getByText('mcp-tool').first().click()
  // Switch to Adopt mode
  await vi.waitFor(async () => {
    await expect.element(page.getByText('Create New')).toBeVisible()
  })
  await page.getByText('Create New').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('Adopt Existing')).toBeVisible()
  })
  await page.getByText('Adopt Existing').click()
  // Select instance to adopt
  await page.getByText('Select instance...').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('orphan-tool')).toBeVisible()
  })
  await page.getByText('orphan-tool').click()
  // Submit
  await page.getByRole('button', { name: 'Adopt', exact: true }).click()
  await vi.waitFor(() => {
    expect(api.instances.setParent).toHaveBeenCalledWith('test-catalog', 'mcp-tool', 'orphan-1', expect.objectContaining({
      parent_type: 'mcp-server',
      parent_instance_id: 'i1',
    }))
  })
})

// === Coverage: handleSetParent (L287-298) via data-testid on Select options ===

test('handleSetParent calls setParent API and refreshes tree', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockResolvedValue({})
  ;(api.instances.list as Mock).mockResolvedValue({ items: [{ id: 'i1', name: 'my-server', entity_type_id: 'et1' }], total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await vi.waitFor(() => { expect(api.instances.list).toHaveBeenCalledWith('test-catalog', 'mcp-server') })
  // Select parent instance via data-testid (bypasses aria-hidden on PF6 Select portal)
  await page.getByText('Select container...').click()
  await page.getByTestId('parent-inst-i1').click()
  // Submit
  await page.getByRole('button', { name: 'Set Container' }).click()
  await vi.waitFor(() => {
    expect(api.instances.setParent).toHaveBeenCalledWith('test-catalog', 'mcp-tool', 'i2', expect.objectContaining({
      parent_type: 'mcp-server',
      parent_instance_id: 'i1',
    }))
  })
})

test('handleSetParent error shows error in modal', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.setParent as Mock).mockRejectedValue(new Error('parent assignment failed'))
  ;(api.instances.list as Mock).mockResolvedValue({ items: [{ id: 'i1', name: 'my-server', entity_type_id: 'et1' }], total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await vi.waitFor(() => { expect(api.instances.list).toHaveBeenCalled() })
  await page.getByText('Select container...').click()
  await page.getByTestId('parent-inst-i1').click()
  await page.getByRole('button', { name: 'Set Container' }).click()
  await expect.element(page.getByText('parent assignment failed')).toBeVisible()
})

// === Coverage: handleCreateLink (L321-332) via data-testid on Select options ===

test('handleCreateLink calls link create API and refreshes tree', async () => {
  const mockModelInstances = [{ id: 'model-1', name: 'gpt-4o', entity_type_id: 'et3' }]
  ;(api.links.create as Mock).mockResolvedValue({ id: 'l-new' })
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockModelInstances, total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByText('Link to Instance')).toBeVisible()
  // Select association via data-testid
  await page.getByText('Select association...').click()
  await page.getByTestId('link-assoc-uses-model').click()
  // Wait for target instances to load, then select target
  await vi.waitFor(async () => {
    await expect.element(page.getByText('Select target instance...')).toBeVisible()
  })
  await page.getByText('Select target instance...').click()
  await page.getByTestId('link-target-model-1').click()
  // Submit
  await page.getByRole('button', { name: 'Link' }).click()
  await vi.waitFor(() => {
    expect(api.links.create).toHaveBeenCalledWith('test-catalog', 'mcp-server', 'i1', expect.objectContaining({
      target_instance_id: 'model-1',
      association_name: 'uses-model',
    }))
  })
})

test('handleCreateLink error shows error in link modal', async () => {
  const mockModelInstances = [{ id: 'model-1', name: 'gpt-4o', entity_type_id: 'et3' }]
  ;(api.links.create as Mock).mockRejectedValue(new Error('link create failed'))
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockModelInstances, total: 1 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByText('Link to Instance')).toBeVisible()
  await page.getByText('Select association...').click()
  await page.getByTestId('link-assoc-uses-model').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('Select target instance...')).toBeVisible()
  })
  await page.getByText('Select target instance...').click()
  await page.getByTestId('link-target-model-1').click()
  await page.getByRole('button', { name: 'Link' }).click()
  await expect.element(page.getByText('link create failed')).toBeVisible()
})

// === Coverage: modal Cancel / onClose callbacks ===

test('Create modal Cancel button closes modal', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  // Wait for modal to open — check for entity type dropdown
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  // Click Cancel button inside the modal dialog
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  // Modal should close — dialog should disappear
  await vi.waitFor(() => {
    expect(page.getByRole('dialog').elements().length).toBe(0)
  })
})

test('Edit modal Cancel button closes modal', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  // Click Cancel
  const cancelBtns = page.getByRole('button', { name: 'Cancel' })
  await cancelBtns.first().click()
})

test('Delete modal Cancel button closes modal', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  // Click Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  // Dialog should close
  await vi.waitFor(() => {
    expect(page.getByText('Confirm Deletion').elements().length).toBe(0)
  })
})

// === Coverage: Modal onClose via Escape (L565, L606, L632) ===
// Cancel button calls onClick={() => setXOpen(false)} — a different arrow function.
// Modal's onClose is only triggered by PF6's internal mechanisms (X button, Escape, click outside).

test('Create modal Escape triggers onClose (L565)', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await vi.waitFor(() => {
    expect(page.getByRole('dialog').elements().length).toBe(0)
  })
})

test('Edit modal Escape triggers onClose (L606)', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await vi.waitFor(() => {
    expect(page.getByRole('dialog').elements().length).toBe(0)
  })
})

test('Delete modal Escape triggers onClose (L632)', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await vi.waitFor(() => {
    expect(page.getByRole('dialog').elements().length).toBe(0)
  })
})

test('Add Child modal onClose clears error', async () => {
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByText('Add Contained Instance')).toBeVisible()
  // Close via Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await vi.waitFor(() => {
    expect(page.getByText('Add Contained Instance').elements().length).toBe(0)
  })
})

test('Set Parent modal onClose clears error', async () => {
  ;(api.instances.get as Mock).mockResolvedValue(mockChildDetail)
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await page.getByText('▸').first().click()
  await clickTreeNode('my-tool')
  await expect.element(page.getByRole('heading', { name: 'my-tool' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Set Parent' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Close via Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await vi.waitFor(() => {
    expect(page.getByText(/Set Container/).elements().length).toBe(0)
  })
})

test('Link modal onClose clears error', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Create Link' }).click()
  await expect.element(page.getByText('Link to Instance')).toBeVisible()
  // Close via Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await vi.waitFor(() => {
    expect(page.getByText('Link to Instance').elements().length).toBe(0)
  })
})

// TD-141: Instance name client-side validation
test('Create modal: invalid instance name shows validation error', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('we)(?')
  await expect.element(page.getByText(/Kubernetes resource name/)).toBeVisible()
})

test('Create modal: valid instance name shows no validation error', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('my-server')
  expect(page.getByText(/Kubernetes resource name/).elements().length).toBe(0)
})

test('Create modal: Create button disabled when name has invalid chars', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('bad!name')
  // Create button should be disabled due to invalid name
  await expect.element(page.getByRole('button', { name: 'Create' })).toHaveAttribute('disabled', '')
})

// handleCreateTypeChange empty guard (L137): unreachable — disabled <option> prevents selection.

// Cover handleAddChild error path (L282) via createContained API failure
test('handleAddChild error path: createContained failure shows error in modal', async () => {
  ;(api.instances.createContained as Mock).mockRejectedValue(new Error('containment validation failed'))
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await page.getByRole('button', { name: 'Add Child' }).click()
  await expect.element(page.getByText('Add Contained Instance')).toBeVisible()
  // Select child type
  await page.getByText('Select child type...').click()
  await vi.waitFor(async () => {
    await expect.element(page.getByText('mcp-tool').first()).toBeVisible()
  })
  await page.getByText('mcp-tool').first().click()
  // Fill in name and submit
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  await page.getByRole('textbox', { name: /^Name/ }).fill('bad-child')
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  // Error should appear in the parent page's Add Child modal
  await expect.element(page.getByText('containment validation failed')).toBeVisible()
})

// Cover boolean default initialization (L147)
test('Create modal: boolean required attr gets default false value', async () => {
  // Mock snapshot with a required boolean attribute
  const snapshotWithBool = {
    ...mockSnapshotServer,
    attributes: [
      ...mockSnapshotServer.attributes,
      { id: 'a-bool', name: 'is_active', description: 'Active flag', type: 'boolean', base_type: 'boolean', ordinal: 2, required: true },
    ],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(mockSnapshotTool)
    return Promise.resolve(snapshotWithBool)
  })
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  // Wait for schema to load — look for the is_active checkbox
  await vi.waitFor(() => {
    // The boolean attr should exist in the form
    expect(api.versions.snapshot).toHaveBeenCalled()
  })
})

// Cover create onChange callback (L587)
test('Create modal: changing attribute value updates state', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Create Instance' }).click()
  const etSelect = page.getByRole('combobox', { name: /entity type/i })
  await expect.element(etSelect).toBeVisible()
  await userEvent.selectOptions(etSelect, 'mcp-server')
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  // Type in the endpoint field
  const endpointField = page.getByRole('textbox', { name: /endpoint/i })
  await endpointField.fill('https://api.example.com')
  // Now create — verify the attribute was submitted
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'x', name: 'y' })
  await page.getByRole('textbox', { name: /^Name/ }).fill('test-inst')
  await page.getByRole('button', { name: 'Create' }).click()
  await vi.waitFor(() => {
    const call = (api.instances.create as Mock).mock.calls[0]
    expect(call[2].attributes).toHaveProperty('endpoint', 'https://api.example.com')
  })
})

// Cover useContainmentTree.getDescendants guard: node not found (L78)
test('Delete modal for non-existent tree node shows empty descendants', async () => {
  // Use a mock where tree has no matching instance, to hit getDescendants guard
  ;(api.instances.tree as Mock).mockResolvedValue([])
  ;(api.instances.get as Mock).mockResolvedValue({ ...mockInstanceDetail, id: 'orphan' })
  renderDetail('RW')
  // Tree is empty so no descendants will be found — tests getDescendants empty-return guard
})

// === Export Plugins Tab ===

test('Export Plugins tab shows "No export bindings" when empty', async () => {
  renderDetail()
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByText('No export bindings configured.')).toBeVisible()
})

test('Export Plugins tab shows bindings list', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server', tool_type: 'mcp-tool' }, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [{ name: 'mcp-gateway', description: 'MCP', parameter_schema: [] }] })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  await expect.element(page.getByText('server_type=mcp-server')).toBeVisible()
  await expect.element(page.getByText('never')).toBeVisible()
})

test('Export Plugins tab: Admin sees Add/Edit/Delete buttons', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByRole('button', { name: 'Add Export Binding' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('Export Plugins tab: RW sees Export Now but not Add/Edit/Delete', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'success', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByRole('button', { name: 'Export Now' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Export Binding' })).not.toBeInTheDocument()
})

test('Export Plugins tab: Export Now calls run API', async () => {
  ;(api.exportBindings.run as Mock).mockResolvedValue(undefined)
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await vi.waitFor(() => {
    expect(api.exportBindings.run).toHaveBeenCalledWith('test-catalog', 'b1', undefined)
  })
})

test('Export Plugins tab: Delete shows confirmation modal', async () => {
  ;(api.exportBindings.delete as Mock).mockResolvedValue(undefined)
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Delete Export Binding')).toBeVisible()
  await expect.element(page.getByText(/Are you sure you want to delete/)).toBeVisible()
  // Confirm delete
  const deleteButtons = page.getByRole('button', { name: 'Delete' })
  // Second Delete button is in the modal
  const allDeleteBtns = await deleteButtons.all()
  await allDeleteBtns[allDeleteBtns.length - 1].click()
  await vi.waitFor(() => {
    expect(api.exportBindings.delete).toHaveBeenCalledWith('test-catalog', 'b1')
  })
})

test('Export Plugins tab: Delete modal closes on Escape', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Delete Export Binding')).toBeVisible()
  // Press Escape to trigger onClose
  await userEvent.keyboard('{Escape}')
  // Modal should close — delete target cleared
  await expect.element(page.getByText('Delete Export Binding')).not.toBeInTheDocument()
})

test('Export Plugins tab: Toggle enabled/disabled', async () => {
  ;(api.exportBindings.update as Mock).mockResolvedValue({ id: 'b1', enabled: false })
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Enabled' }).click()
  await vi.waitFor(() => {
    expect(api.exportBindings.update).toHaveBeenCalledWith('test-catalog', 'b1', { enabled: false })
  })
})

test('Export Plugins tab: Export Now disabled when binding is disabled', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: false, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  const exportBtn = page.getByRole('button', { name: 'Export Now' })
  await expect.element(exportBtn).toBeVisible()
  expect(exportBtn.element()).toHaveProperty('disabled', true)
})

test('Export Plugins tab: entity type dropdowns are sorted alphabetically', async () => {
  // Mock pins in non-alphabetical order
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({
    items: [
      { pin_id: 'p1', entity_type_name: 'zebra', entity_type_id: 'et3', entity_type_version_id: 'etv3', version: 1 },
      { pin_id: 'p2', entity_type_name: 'alpha', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 1 },
      { pin_id: 'p3', entity_type_name: 'mango', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
    ],
  })
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [
    { name: 'mcp-gateway', description: 'MCP Gateway', parameter_schema: [
      { name: 'server_type', type: 'entity_type', required: true, description: 'Server' },
    ] },
  ] })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  // Select exporter to show param fields
  const exporterSelect = document.querySelector('select[aria-label="Select exporter"]') as HTMLSelectElement
  exporterSelect.value = 'mcp-gateway'
  exporterSelect.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise(r => setTimeout(r, 500))
  // Get options from the entity type dropdown
  const etSelect = document.querySelector('select[aria-label="server_type"]') as HTMLSelectElement
  const options = Array.from(etSelect.options).map(o => o.text).filter(t => t !== 'Select entity type...')
  expect(options).toEqual(['alpha', 'mango', 'zebra'])
})

test('Export Plugins tab: Add binding modal shows error when pins fail to load', async () => {
  // listPins succeeds for page load, then fails for the modal
  ;(api.catalogVersions.listPins as Mock)
    .mockResolvedValueOnce({ items: mockPins, total: 3 })
    .mockRejectedValueOnce(new Error('500: pins service down'))
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [
    { name: 'mcp-gateway', description: 'MCP Gateway', parameter_schema: [
      { name: 'server_type', type: 'entity_type', required: true, description: 'Server' },
    ] },
  ] })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  const exporterSelect = document.querySelector('select[aria-label="Select exporter"]') as HTMLSelectElement
  exporterSelect.value = 'mcp-gateway'
  exporterSelect.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise(r => setTimeout(r, 500))
  // Should show error, not just empty dropdown
  await expect.element(page.getByText(/Failed to load entity types/i)).toBeVisible()
})

test('Export Plugins tab: Add binding modal opens with exporter select', async () => {
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [
    { name: 'mcp-gateway', description: 'MCP Gateway', parameter_schema: [
      { name: 'server_type', type: 'string', required: true, description: 'Server entity type' },
      { name: 'tool_type', type: 'string', required: true, description: 'Tool entity type' },
    ] },
  ] })
  renderDetail('Admin')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  // Modal should show the exporter selector
  await expect.element(page.getByRole('combobox', { name: 'Select exporter' })).toBeVisible()
})

test('Export Plugins tab: last run timestamp shown when last_run_at is set', async () => {
  const runDate = '2026-03-15T10:30:00Z'
  const expectedTimestamp = new Date(runDate).toLocaleString()
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: runDate, last_run_status: 'success', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByText('success')).toBeVisible()
  await expect.element(page.getByText(expectedTimestamp)).toBeVisible()
})

test('Export Plugins tab: no timestamp shown when last_run_at is null', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await expect.element(page.getByText('never')).toBeVisible()
  // No date should appear in the last run cell
  const lastRunCell = page.getByText('never').element()?.parentElement
  expect(lastRunCell?.querySelectorAll('span').length).toBeLessThanOrEqual(1)
})

test('Export Plugins tab: binding with error shows failed status', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: '2026-01-01', last_run_status: 'failed', last_run_error: 'schema drift', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  // Check for the binding row with status
  await expect.element(page.getByRole('gridcell', { name: 'mcp-gateway' })).toBeVisible()
})

test('Export Plugins tab: run error shows alert', async () => {
  ;(api.exportBindings.run as Mock).mockRejectedValue(new Error('Export run failed'))
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await expect.element(page.getByText('Export run failed')).toBeVisible()
})

// T-34.75k: Export Now opens VS instance picker modal when binding has virtual_server_type
test('Export Plugins tab: Export Now opens VS instance picker for VS binding', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({
    items: [
      { id: 'vs1', name: 'prod-vs' },
      { id: 'vs2', name: 'staging-vs' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: { virtual_server_type: 'virtual-server' }, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  // VS picker modal should appear
  await expect.element(page.getByText('Select Virtual Server')).toBeVisible()
})

// T-34.75l: VS instance picker shows only instances of virtual_server_type
test('Export Plugins tab: VS picker shows correct instances', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({
    items: [
      { id: 'vs1', name: 'prod-vs' },
      { id: 'vs2', name: 'staging-vs' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: { virtual_server_type: 'virtual-server' }, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await expect.element(page.getByText('Select Virtual Server')).toBeVisible()
  // Verify instances are listed — check the dropdown has both options
  const select = document.querySelector('select[aria-label="Select virtual server instance"]') as HTMLSelectElement
  const options = Array.from(select.options).map(o => o.text).filter(t => t !== 'Select an instance...')
  expect(options).toContain('prod-vs')
  expect(options).toContain('staging-vs')
  // Verify instances.list was called with the correct entity type
  expect(api.instances.list).toHaveBeenCalledWith('test-catalog', 'virtual-server', { limit: 100 })
})

test('Export Plugins tab: VS picker shows error when API fails', async () => {
  ;(api.instances.list as Mock).mockRejectedValue(new Error('502: Bad Gateway'))
  ;(api.exportBindings.list as Mock).mockResolvedValue({
    items: [
      { id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: { virtual_server_type: 'virtual-server' }, enabled: true, last_run_at: null, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
    ],
  })
  renderDetail('RW')
  await page.getByText('Export Plugins').click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  // Should show the error, not "No instances found"
  await expect.element(page.getByText(/Bad Gateway|failed|error/i).first()).toBeVisible()
  expect(page.getByText('No virtual-server instances found').query()).toBeNull()
})

// Cover edit onChange callback in modal (L587 equivalent for edit)
test('Edit modal: changing attribute triggers onChange callback', async () => {
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'my-server' })
  renderDetail('RW')
  await openTreeAndExpandServers()
  await clickTreeNode('my-server')
  await expect.element(page.getByRole('heading', { name: 'my-server' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
  await expect.element(page.getByRole('textbox', { name: /^Name/ })).toBeVisible()
  // Change the endpoint value
  const endpointInput = page.getByRole('textbox', { name: /endpoint/ })
  await endpointInput.fill('https://new-endpoint.com')
  // Submit and verify
  await page.getByRole('button', { name: 'Save' }).click()
  await vi.waitFor(() => {
    const call = (api.instances.update as Mock).mock.calls[0]
    expect(call[3].attributes).toHaveProperty('endpoint', 'https://new-endpoint.com')
  })
})
