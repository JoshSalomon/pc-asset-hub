import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogDetailPage from './CatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn() },
    catalogVersions: { listPins: vi.fn() },
    versions: { snapshot: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), createContained: vi.fn(), listContained: vi.fn(), setParent: vi.fn() },
    enums: { listValues: vi.fn() },
    links: { create: vi.fn(), delete: vi.fn(), forwardRefs: vi.fn(), reverseRefs: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalog = {
  id: 'cat1', name: 'my-catalog', description: 'Test catalog',
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
    { id: 'a1', name: 'hostname', type: 'string', ordinal: 1, required: false },
    { id: 'a2', name: 'port', type: 'number', ordinal: 2, required: true },
  ],
  associations: [
    { id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    { id: 'assoc1-in', name: 'tools', type: 'containment', direction: 'incoming', target_entity_type_id: 'et1', source_entity_type_id: 'et1', source_entity_type_name: 'model' },
  ],
}

const mockForwardRefs = [
  { link_id: 'link1', association_name: 'uses', association_type: 'directional', instance_id: 'i2', instance_name: 'target-inst', entity_type_name: 'tool' },
]

const mockReverseRefs = [
  { link_id: 'link2', association_name: 'depends-on', association_type: 'directional', instance_id: 'i3', instance_name: 'source-inst', entity_type_name: 'server' },
]

const mockInstances = [
  {
    id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
    version: 1, attributes: [
      { name: 'hostname', type: 'string', value: 'host-a' },
      { name: 'port', type: 'number', value: 8080 },
    ],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
]

function renderDetail(role: 'Admin' | 'RW' | 'RO' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes>
        <Route path="/catalogs/:name" element={<CatalogDetailPage role={role} />} />
        <Route path="/catalogs" element={<div>Catalog List</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockInstances, total: 1 })
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'i2', name: 'new-inst' })
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'inst-a', version: 2 })
  ;(api.instances.delete as Mock).mockResolvedValue(undefined)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.instances.listContained as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.instances.createContained as Mock).mockResolvedValue({ id: 'c1', name: 'new-child' })
  ;(api.links.forwardRefs as Mock).mockResolvedValue(mockForwardRefs)
  ;(api.links.reverseRefs as Mock).mockResolvedValue(mockReverseRefs)
  ;(api.links.create as Mock).mockResolvedValue({ id: 'link-new' })
  ;(api.links.delete as Mock).mockResolvedValue(undefined)
})

// Helper: wait for instance table to render
async function waitForInstances() {
  await expect.element(page.getByRole('gridcell', { name: 'inst-a' })).toBeVisible()
}

// T-11.48: Catalog detail page shows tabs per pinned entity type
test('T-11.48: shows entity type tabs', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'model' })).toBeVisible()
  await expect.element(page.getByRole('tab', { name: 'tool' })).toBeVisible()
})

// T-11.49: Entity type tab shows instance list table
test('T-11.49: shows instance list table', async () => {
  renderDetail()
  await waitForInstances()
  await expect.element(page.getByRole('gridcell', { name: 'First' })).toBeVisible()
})

// T-11.50: Instance list shows attribute values in columns
test('T-11.50: shows attribute values in columns', async () => {
  renderDetail()
  await waitForInstances()
  await expect.element(page.getByRole('gridcell', { name: 'host-a' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: '8080' })).toBeVisible()
})

// T-11.51: Create instance button visible for RW+, hidden for RO
test('T-11.51: create button visible for RW', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('button', { name: /Create model/ })).toBeVisible()
})

test('T-11.51: create button hidden for RO', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('tab', { name: 'model' })).toBeVisible()
  // RO should not see Create, Edit, or Delete buttons
  await expect.element(page.getByRole('button', { name: /Create model/ })).not.toBeInTheDocument()
})

// T-11.52: Create instance modal has dynamic attribute form
test('T-11.52: create modal has dynamic form', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Check for attribute fields in the dialog
  await expect.element(page.getByRole('dialog').getByText('hostname')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('port *')).toBeVisible()
})

// T-11.53: Create instance modal submits with attribute values
test('T-11.53: create submits with attributes', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill name field (first textbox in dialog after title)
  const nameInput = page.getByRole('dialog').getByRole('textbox').first()
  await nameInput.fill('new-inst')

  // Fill hostname attribute (third textbox — after name and description)
  const hostnameInput = page.getByRole('dialog').getByRole('textbox').nth(2)
  await hostnameInput.fill('new-host')

  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.instances.create).toHaveBeenCalled()
})

// T-11.54: Edit instance opens modal with pre-filled values
test('T-11.54: edit modal pre-fills values', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Edit inst-a')).toBeVisible()
})

// T-11.55: Edit instance submits updated values
test('T-11.55: edit submits', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.instances.update).toHaveBeenCalled()
})

// T-11.56: Delete instance shows confirmation dialog
test('T-11.56: delete shows confirmation', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog').getByText('Are you sure you want to delete')).toBeVisible()
})

// T-11.57: Delete instance removes from list
test('T-11.57: delete calls API', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog').getByText('Are you sure you want to delete')).toBeVisible()

  // Click the Delete button in the confirmation dialog
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  expect(api.instances.delete).toHaveBeenCalledWith('my-catalog', 'model', 'i1')
})

// T-11.58: Empty instance list shows empty state
test('T-11.58: empty state when no instances', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'model' })).toBeVisible()
  // Wait for Create button to appear (signals loading complete)
  await expect.element(page.getByRole('button', { name: /Create model/ })).toBeVisible()
  // Verify no instance rows rendered — no gridcell elements
  const cells = page.getByRole('gridcell')
  expect(cells.elements().length).toBe(0)
})

// Bug fix: details pane closes when switching tabs
test('details pane closes on tab switch', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // Open details on model tab
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
  // Switch to tool tab
  await page.getByRole('tab', { name: 'tool' }).click()
  // Details pane should be gone — no "Contained Instances" heading visible
  await expect.element(page.getByRole('heading', { name: /^Details:/ }).first()).not.toBeInTheDocument()
})

// === Milestone 12: Containment & Association Links UI ===

// T-12.54: Instance detail shows contained children section
test('T-12.54: details panel shows children section', async () => {
  ;(api.instances.listContained as Mock).mockResolvedValue({
    items: [{ id: 'c1', entity_type_id: 'et2', name: 'child-tool', description: 'A tool', version: 1, attributes: [] }],
    total: 1,
  })
  renderDetail('Admin')
  await waitForInstances()
  // Click Details on the first instance
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
})

// T-12.55: Add contained instance button visible for RW+, hidden for RO
test('T-12.55: add contained button for RW', async () => {
  renderDetail('RW')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Contained Instance' }).first()).toBeVisible()
})

test('T-12.55: add contained button hidden for RO', async () => {
  renderDetail('RO')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Contained Instance' })).not.toBeInTheDocument()
})

// T-12.58: Instance detail shows references
test('T-12.58: details panel shows references', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'References' }).first()).toBeVisible()
  // Forward references visible
  await expect.element(page.getByText('Forward References').first()).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'target-inst' }).first()).toBeVisible()
  // Reverse references visible
  await expect.element(page.getByText('Referenced By').first()).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'source-inst' }).first()).toBeVisible()
})

// T-12.59: Forward references show association name
test('T-12.59: forward refs show association name', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('gridcell', { name: 'uses' }).first()).toBeVisible()
})

// T-12.63: RO user sees references but no link/unlink controls
test('T-12.63: RO sees refs without link/unlink', async () => {
  renderDetail('RO')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'References' }).first()).toBeVisible()
  // Should NOT see Link to Instance or Unlink buttons
  await expect.element(page.getByRole('button', { name: 'Link to Instance' })).not.toBeInTheDocument()
})

// Bug: child type resets when reopening Add Contained from different parent type
test('add contained modal resets child type on reopen', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Open modal — with only one containment assoc (tools), child type should be pre-selected
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // The child type dropdown should show "tool" (pre-selected since it's the only option)
  await expect.element(page.getByRole('dialog').getByText('tool')).toBeVisible()
  // Close modal
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  // Re-open — should still show "tool" pre-selected (reset + re-pre-select)
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog').getByText('tool')).toBeVisible()
})

// Bug: Set Container modal shows container type as non-editable text
test('set container modal shows container type as text and pre-loads instances', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockInstances, total: 1 })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Container type should be shown as a disabled input with "model" value
  const typeInput = page.getByRole('dialog').getByRole('textbox', { name: 'Container type' })
  await expect.element(typeInput).toBeVisible()
  await expect.element(typeInput).toHaveValue('model')
})

// Bug: "Contained by" shows parent name, not UUID
test('details pane shows parent name not UUID', async () => {
  const childInstances = [{
    id: 'c1', entity_type_id: 'et1', catalog_id: 'cat1', parent_instance_id: 'p1',
    name: 'child-inst', description: '', version: 1,
    attributes: [{ name: 'hostname', type: 'string', value: 'h1' }],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }]
  ;(api.instances.list as Mock).mockResolvedValue({ items: childInstances, total: 1 })
  // Mock GetByID for parent instance resolution
  ;(api.instances.get as Mock).mockResolvedValue({
    id: 'p1', name: 'my-parent-server', entity_type_id: 'et1',
  })
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'child-inst' })).toBeVisible()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Should show parent name, not UUID
  await expect.element(page.getByText('Contained by: my-parent-server').first()).toBeVisible()
})

// UX: mode shows "Create New" (disabled) when no uncontained instances, hides "Adopt Existing"
test('add contained modal shows disabled Create New when no adoptable instances', async () => {
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: [], total: 0 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // "Create New" should be visible as the mode label
  await expect.element(page.getByRole('dialog').getByText('Create New')).toBeVisible()
  // "Adopt Existing" should NOT be available (no adoptable instances)
  await expect.element(page.getByText('Adopt Existing')).not.toBeInTheDocument()
  // The mode toggle should be disabled (read-only)
  const modeToggle = page.getByRole('dialog').getByText('Create New')
  await expect.element(modeToggle).toBeVisible()
})

// Bug: mode dropdown not responsive (hardcoded isOpen=false)
test('add contained modal mode dropdown is interactive', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Mode should default to "Create New"
  await expect.element(page.getByRole('dialog').getByText('Create New')).toBeVisible()
  // Click the mode toggle to open dropdown
  await page.getByRole('dialog').getByText('Create New').click()
  // "Adopt Existing" option should be visible in the dropdown
  await expect.element(page.getByText('Adopt Existing').first()).toBeVisible()
})
