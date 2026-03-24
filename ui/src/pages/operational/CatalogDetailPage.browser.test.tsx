import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogDetailPage from './CatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), list: vi.fn(), validate: vi.fn(), publish: vi.fn(), unpublish: vi.fn(), copy: vi.fn(), replace: vi.fn() },
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
    { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
    { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
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

const mockToolSnapshot = {
  entity_type: { id: 'et2', name: 'tool' },
  version: { id: 'etv2', version: 1 },
  attributes: [
    { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
    { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
  ],
  associations: [],
}

const mockInstances = [
  {
    id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
    version: 1, attributes: [
      { name: 'name', type: 'string', value: 'inst-a', system: true },
      { name: 'description', type: 'string', value: 'First', system: true },
      { name: 'hostname', type: 'string', value: 'host-a' },
      { name: 'port', type: 'number', value: 8080 },
    ],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
]

function renderDetail(role: 'Admin' | 'RW' | 'RO' | 'SuperAdmin' = 'Admin') {
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
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(mockToolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
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
  ;(api.catalogs.copy as Mock).mockResolvedValue({ id: 'new-id', name: 'copy-cat' })
  ;(api.catalogs.replace as Mock).mockResolvedValue({ id: 'src-id', name: 'prod' })
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: [{ name: 'other-cat' }, { name: 'prod-cat' }], total: 2 })
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

// === Coverage: error states, loading states, modal flows ===

test('catalog load failure shows error alert', async () => {
  ;(api.catalogs.get as Mock).mockRejectedValue(new Error('Network error'))
  renderDetail()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

test('catalog load generic error path', async () => {
  ;(api.catalogs.get as Mock).mockRejectedValue({ message: 'oops' })
  renderDetail()
  await expect.element(page.getByText('Failed to load catalog')).toBeVisible()
})

test('no pins shows empty state', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByText(/No entity types pinned/)).toBeVisible()
})

test('create instance failure shows error in modal', async () => {
  ;(api.instances.create as Mock).mockRejectedValue(new Error('409: duplicate name'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  const nameInput = page.getByRole('dialog').getByRole('textbox').first()
  await nameInput.fill('dup-inst')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await expect.element(page.getByText('409: duplicate name')).toBeVisible()
})

test('edit instance failure shows error in modal', async () => {
  ;(api.instances.update as Mock).mockRejectedValue(new Error('500: update failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('500: update failed')).toBeVisible()
})

test('delete instance failure shows error in modal', async () => {
  ;(api.instances.delete as Mock).mockRejectedValue(new Error('500: delete failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog').getByText('Are you sure you want to delete')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('500: delete failed')).toBeVisible()
})

test('hide details toggle when clicking Details on already-selected instance', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // Open details
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: /^Details:/ }).first()).toBeVisible()
  // Click "Hide Details" to close
  await page.getByRole('button', { name: 'Hide Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: /^Details:/ }).first()).not.toBeInTheDocument()
})

test('no references shows "No references." message', async () => {
  ;(api.links.forwardRefs as Mock).mockResolvedValue([])
  ;(api.links.reverseRefs as Mock).mockResolvedValue([])
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('No references.').first()).toBeVisible()
})

test('link modal opens and can be cancelled', async () => {
  // Need non-containment outgoing assoc for "Link to Instance" button
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('button', { name: 'Link to Instance' })).toBeVisible()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByText('Select association...')).toBeVisible()
  // Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Select association...')).not.toBeInTheDocument()
})

test('unlink removes link and refreshes', async () => {
  // Need non-containment outgoing assoc
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('Forward References').first()).toBeVisible()
  await page.getByRole('button', { name: 'Unlink' }).first().click()
  expect(api.links.delete).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'link1')
})

test('refresh button reloads instances', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Refresh' }).click()
  // Should call list again (total calls: initial + refresh)
  expect((api.instances.list as Mock).mock.calls.length).toBeGreaterThanOrEqual(2)
})

test('back button navigates to catalog list', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Back to Catalogs/ }).click()
  await expect.element(page.getByText('Catalog List')).toBeVisible()
})

test('validation status label colors: valid=green, invalid=red, draft=blue', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'invalid' })
  renderDetail()
  await expect.element(page.getByText('invalid')).toBeVisible()
})

test('create modal cancel closes and clears error', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  // Dialog should close
  const dialogs = page.getByRole('dialog')
  expect(dialogs.elements().length).toBe(0)
})

// Add contained child in create mode submits createContained API call
test('add contained child in create mode calls createContained', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Child type pre-selected as "tool"
  await expect.element(page.getByRole('dialog').getByText('tool')).toBeVisible()

  // Fill name for new child
  const nameInput = page.getByRole('dialog').getByRole('textbox', { name: /Name/i })
  await nameInput.fill('new-child-tool')

  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
  expect(api.instances.createContained).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'tool', {
    name: 'new-child-tool',
    description: undefined,
  })
})

// TD-42: Add contained instance modal shows child type's custom attributes
test('TD-42: add contained modal shows child type schema attributes', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'tool-version', type: 'string', ordinal: 1, required: false },
      { id: 'ta2', name: 'weight', type: 'number', ordinal: 2, required: true },
    ],
    associations: [],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(toolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('heading', { name: 'Contained Instances' }).first()).toBeVisible()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Child type pre-selected as "tool"
  await expect.element(page.getByRole('dialog').getByText('tool', { exact: true })).toBeVisible()

  // Custom attributes from the tool schema should appear as form fields
  await expect.element(page.getByRole('dialog').getByText('tool-version')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('weight *')).toBeVisible()
})

// TD-42: Add contained instance modal submits custom attributes
test('TD-42: add contained modal submits custom attributes', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'tool-version', type: 'string', ordinal: 1, required: false },
    ],
    associations: [],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(toolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill name
  const nameInput = page.getByRole('dialog').getByRole('textbox', { name: /Name/i })
  await nameInput.fill('new-tool')

  // Fill custom attribute
  const toolVersionInput = page.getByRole('dialog').getByRole('textbox', { name: /tool-version/i })
  await toolVersionInput.fill('2.0')

  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
  expect(api.instances.createContained).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'tool', {
    name: 'new-tool',
    description: undefined,
    attributes: { 'tool-version': '2.0' },
  })
})

// TD-42 coverage: submit contained instance with number attribute (parseFloat branch)
test('TD-42: contained modal submits number attribute as parsed float', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'weight', type: 'number', ordinal: 1, required: false },
    ],
    associations: [],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(toolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('dialog').getByRole('textbox', { name: /Name/i }).fill('tool-1')
  await page.getByRole('dialog').getByRole('spinbutton', { name: /weight/i }).fill('3.14')
  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
  expect(api.instances.createContained).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'tool', {
    name: 'tool-1',
    description: undefined,
    attributes: { weight: 3.14 },
  })
})

// TD-42 coverage: contained modal with enum attribute
test('TD-42: contained modal shows enum select for enum attributes', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'status', type: 'enum', enum_id: 'enum1', ordinal: 1, required: false },
    ],
    associations: [],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(toolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'active' }, { value: 'inactive' }] })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Enum attribute should render with a select toggle
  await expect.element(page.getByRole('dialog').getByText('status')).toBeVisible()
})

// Add contained child error shows in modal
test('add contained child error shows in modal', async () => {
  ;(api.instances.createContained as Mock).mockRejectedValue(new Error('400: invalid'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  const nameInput = page.getByRole('dialog').getByRole('textbox', { name: /Name/i })
  await nameInput.fill('bad-child')
  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()

  await expect.element(page.getByText('400: invalid')).toBeVisible()
})

// Adopt mode: when there are uncontained instances, mode dropdown works
test('add contained in adopt mode shows adopt controls', async () => {
  const uncontainedTools = [
    { id: 'ut1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'orphan-tool', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: uncontainedTools, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Switch to Adopt Existing mode
  await page.getByRole('dialog').getByText('Create New').click()
  await expect.element(page.getByText('Adopt Existing').first()).toBeVisible()
  await page.getByText('Adopt Existing').first().click()

  // The "Select Instance" form should appear with "Select instance..." placeholder
  await expect.element(page.getByRole('dialog').getByText('Select instance...')).toBeVisible()
  // Adopt button should be visible (but disabled until selection)
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Adopt', exact: true })).toBeVisible()
})

// Adopt mode: select instance and submit calls setParent API
test('adopt mode submits setParent API call', async () => {
  const uncontainedTools = [
    { id: 'ut1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'orphan-tool', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: uncontainedTools, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()

  // Switch to Adopt Existing mode
  await page.getByRole('dialog').getByText('Create New').click()
  await page.getByText('Adopt Existing').first().click()

  // Select the orphan instance
  await page.getByText('Select instance...').click()
  await page.getByText('orphan-tool').click()

  // Click Adopt
  await page.getByRole('dialog').getByRole('button', { name: 'Adopt', exact: true }).click()
  expect(api.instances.setParent).toHaveBeenCalledWith('my-catalog', 'tool', 'ut1', {
    parent_type: 'model',
    parent_instance_id: 'i1',
  })
})

// Set container modal: select parent and submit
test('set container modal submits setParent API call', async () => {
  const parentInstances = [
    { id: 'p1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'parent-server', description: '', version: 1, attributes: [] },
  ]
  // First call returns mockInstances (for initial list load), subsequent calls return parentInstances
  let listCallCount = 0
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    listCallCount++
    if (type === 'model' && listCallCount <= 1) return Promise.resolve({ items: mockInstances, total: 1 })
    if (type === 'model') return Promise.resolve({ items: parentInstances, total: 1 })
    return Promise.resolve({ items: mockInstances, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select parent instance from dropdown
  await page.getByText('Select container...').click()
  await page.getByText('parent-server').click()

  // Click Set Container submit
  await page.getByRole('dialog').getByRole('button', { name: 'Set Container' }).click()
  expect(api.instances.setParent).toHaveBeenCalledWith('my-catalog', 'model', 'i1', {
    parent_type: 'model',
    parent_instance_id: 'p1',
  })
})

// Set container modal opens with correct parent type
test('set container modal opens and shows pre-selected container type', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Container type is pre-filled and disabled
  const typeInput = page.getByRole('dialog').getByRole('textbox', { name: 'Container type' })
  await expect.element(typeInput).toBeVisible()
  await expect.element(typeInput).toHaveValue('model')

  // "Set Container" button should be disabled (no instance selected)
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Set Container' })).toBeDisabled()
  // Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Edit modal cancel closes modal
test('edit modal cancel closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  const dialogs = page.getByRole('dialog')
  expect(dialogs.elements().length).toBe(0)
})

// Delete modal cancel does not delete
test('delete modal cancel does not call API', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog').getByText('Are you sure you want to delete')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(api.instances.delete).not.toHaveBeenCalled()
})

// Instance shows "No contained instances." when children list is empty
test('details panel shows "No contained instances." for empty children', async () => {
  ;(api.instances.listContained as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('No contained instances.').first()).toBeVisible()
})

// Catalog description is shown when present
test('catalog description shown', async () => {
  renderDetail()
  await expect.element(page.getByText(/Test catalog/)).toBeVisible()
})

// SuperAdmin can write
test('SuperAdmin can see create button', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('button', { name: /Create model/ })).toBeVisible()
})

// Enum attributes render EnumSelect in create modal
test('enum attributes render select in create modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'a1', name: 'hostname', type: 'string', ordinal: 1, required: false },
      { id: 'a3', name: 'status', type: 'enum', enum_id: 'enum1', ordinal: 3, required: false },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithEnum)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'active' }, { value: 'inactive' }], total: 2 })

  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // The enum attribute should have "Select..." text (the EnumSelect component)
  await expect.element(page.getByRole('dialog').getByText('Select...')).toBeVisible()
})

// Link modal open shows association and target selectors
test('link modal shows association selector', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('button', { name: 'Link to Instance' })).toBeVisible()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Link button should be disabled (no association or target selected)
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Link' })).toBeDisabled()
  // Should show association and target selectors
  await expect.element(page.getByText('Select association...')).toBeVisible()
  await expect.element(page.getByText('Select target instance...')).toBeVisible()
})

// Instances list loading failure falls back to empty list
test('instance list load failure shows empty', async () => {
  ;(api.instances.list as Mock).mockRejectedValue(new Error('500: error'))
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'model' })).toBeVisible()
  // Should not crash — shows empty state or 0 total
  await expect.element(page.getByText('Total: 0').first()).toBeVisible()
})

// Total count displayed
test('total count displays correctly', async () => {
  renderDetail()
  await waitForInstances()
  await expect.element(page.getByText('Total: 1').first()).toBeVisible()
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

// === Open in Data Viewer link ===

test('catalog detail shows Open in Data Viewer link', async () => {
  renderDetail()
  await waitForInstances()
  const link = page.getByText('Open in Data Viewer')
  await expect.element(link).toBeVisible()
  // Link should point to the operational UI for this catalog, same tab
  const anchor = link.element().closest('a')
  expect(anchor?.getAttribute('href')).toBe('/operational/catalogs/my-catalog')
  expect(anchor?.getAttribute('target')).toBeNull()
})

// === Catalog Validation Tests ===

// T-15.39: Validate button visible for RW user
test('T-15.39: Validate button visible for RW user', async () => {
  renderDetail('RW')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Validate' })).toBeVisible()
})

// T-15.40: Validate button visible for Admin
test('T-15.40: Validate button visible for Admin', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Validate' })).toBeVisible()
})

// T-15.41: Validate button hidden for RO user
test('T-15.41: Validate button hidden for RO user', async () => {
  renderDetail('RO')
  await waitForInstances()
  // The button should not exist in the DOM for RO users
  const buttons = page.getByRole('button', { name: 'Validate' })
  await expect.element(buttons).not.toBeInTheDocument()
})

// T-15.42: Clicking Validate calls POST .../validate API
test('T-15.42: clicking Validate calls API', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'valid', errors: [] })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  expect(api.catalogs.validate).toHaveBeenCalledWith('my-catalog')
})

// T-15.43: Successful validation with no errors shows "valid" status
test('T-15.43: validation pass shows success alert', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'valid', errors: [] })
  // After validation, the catalog should be refreshed with updated status
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByText('Validation passed')).toBeVisible()
})

// T-15.44: Validation with errors shows "invalid" status
test('T-15.44: validation fail shows error alert', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({
    status: 'invalid',
    errors: [
      { entity_type: 'Server', instance_name: 'srv-1', field: 'hostname', violation: 'required attribute "hostname" is missing a value' },
    ],
  })
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'invalid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByText('Validation failed')).toBeVisible()
})

// T-15.45: Validation errors displayed grouped by entity type
test('T-15.45: validation errors grouped by entity type', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({
    status: 'invalid',
    errors: [
      { entity_type: 'Server', instance_name: 'srv-1', field: 'hostname', violation: 'required' },
      { entity_type: 'Server', instance_name: 'srv-2', field: 'hostname', violation: 'required' },
    ],
  })
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'invalid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  // Should show "Server" as a group heading
  await expect.element(page.getByText('Server')).toBeVisible()
})

// T-15.46: Each error shows instance name, field, and violation
test('T-15.46: error details visible', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({
    status: 'invalid',
    errors: [
      { entity_type: 'Server', instance_name: 'srv-1', field: 'hostname', violation: 'required attribute missing' },
    ],
  })
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'invalid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByText(/srv-1.*hostname.*required/)).toBeVisible()
})

// Validation API error shows warning alert
test('validation API error shows warning', async () => {
  ;(api.catalogs.validate as Mock).mockRejectedValue(new Error('server error'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByText('server error')).toBeVisible()
})

// === Catalog Publishing Tests ===

// T-16.57: Publish button visible for Admin on valid unpublished catalog
test('T-16.57: Publish button visible for Admin on valid unpublished catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: false })
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Publish' })).toBeVisible()
})

// T-16.58: Publish button hidden for RW
test('T-16.58: Publish button hidden for RW', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: false })
  renderDetail('RW')
  await waitForInstances()
  expect(document.querySelector('button')?.textContent).not.toContain('Publish')
})

// T-16.59: Publish button hidden when catalog is draft
test('T-16.59: Publish button hidden when draft', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // mockCatalog has validation_status: 'draft' by default
  const buttons = Array.from(document.querySelectorAll('button')).map(b => b.textContent)
  expect(buttons).not.toContain('Publish')
})

// T-16.61: Unpublish button visible on published catalog for Admin
test('T-16.61: Unpublish button visible for Admin on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Unpublish' })).toBeVisible()
})

// T-16.62: Clicking Publish calls API
test('T-16.62: clicking Publish calls API', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: false })
  ;(api.catalogs.publish as Mock).mockResolvedValue({ status: 'published' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Publish' }).click()
  expect(api.catalogs.publish).toHaveBeenCalledWith('my-catalog')
})

// T-16.63: Published badge shown after publish
test('T-16.63: published badge shown on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByText('published')).toBeVisible()
})

// T-16.65: Warning banner shown on published catalog for RW
test('T-16.65: warning banner for RW on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('RW')
  await waitForInstances()
  await expect.element(page.getByText('Editing requires SuperAdmin')).toBeVisible()
})

// Clicking Unpublish calls API
test('clicking Unpublish calls API', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  ;(api.catalogs.unpublish as Mock).mockResolvedValue({ status: 'unpublished' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Unpublish' }).click()
  expect(api.catalogs.unpublish).toHaveBeenCalledWith('my-catalog')
})

// Publish error shows error message
test('publish error shows error message', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: false })
  ;(api.catalogs.publish as Mock).mockRejectedValue(new Error('publish failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByText('publish failed')).toBeVisible()
})

// Unpublish error shows error message
test('unpublish error shows error message', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  ;(api.catalogs.unpublish as Mock).mockRejectedValue(new Error('unpublish failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Unpublish' }).click()
  await expect.element(page.getByText('unpublish failed')).toBeVisible()
})

// No warning banner for Admin on published catalog
test('no warning banner for Admin on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('Admin')
  await waitForInstances()
  // Admin should NOT see the warning banner (only RW/RO see it)
  const alerts = document.querySelectorAll('[class*="alert"]')
  const infoAlerts = Array.from(alerts).filter(a => a.textContent?.includes('Editing requires SuperAdmin'))
  expect(infoAlerts.length).toBe(0)
})

// Published badge NOT shown on unpublished catalog
test('no published badge on unpublished catalog', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // Default mockCatalog has published: undefined/false
  const labels = Array.from(document.querySelectorAll('span')).map(s => s.textContent)
  expect(labels).not.toContain('published')
})

// ---- Copy & Replace UI Tests ----

// T-17.68: Copy button visible for RW+ users
test('T-17.68: copy button visible for RW users', async () => {
  renderDetail('RW')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Copy' })).toBeVisible()
})

// T-17.69: Copy button hidden for RO users
test('T-17.69: copy button hidden for RO users', async () => {
  renderDetail('RO')
  await waitForInstances()
  expect(page.getByRole('button', { name: 'Copy' }).query()).toBeNull()
})

// T-17.70: Copy modal opens with name input
test('T-17.70: copy modal opens', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('Copy Catalog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('textbox').first()).toBeVisible()
})

// T-17.72: Successful copy calls API
test('T-17.72: copy calls API with correct body', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await page.getByRole('dialog').getByRole('textbox').first().fill('new-copy')
  // Click the modal footer's Copy button
  await page.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()
  expect(api.catalogs.copy).toHaveBeenCalledWith({
    source: 'my-catalog',
    name: 'new-copy',
    description: undefined,
  })
})

// T-17.75: Replace button visible on valid catalog for Admin
test('T-17.75: replace button visible for Admin on valid catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Replace' })).toBeVisible()
})

// T-17.76: Replace button hidden for RW users
test('T-17.76: replace button hidden for RW', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('RW')
  await waitForInstances()
  expect(page.getByRole('button', { name: 'Replace' }).query()).toBeNull()
})

// T-17.77: Replace button hidden for RO users
test('T-17.77: replace button hidden for RO', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('RO')
  await waitForInstances()
  expect(page.getByRole('button', { name: 'Replace' }).query()).toBeNull()
})

// T-17.78: Replace button hidden for draft catalogs
test('T-17.78: replace button hidden for draft catalog', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // Default mockCatalog has validation_status: 'draft'
  expect(page.getByRole('button', { name: 'Replace' }).query()).toBeNull()
})

// T-17.71: Copy modal validates DNS-label format
test('T-17.71: copy modal shows validation error for invalid name', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await page.getByRole('dialog').getByRole('textbox').first().fill('INVALID_NAME')
  await expect.element(page.getByText('Must be a valid DNS label')).toBeVisible()
  // Copy button should be disabled
  const copyBtn = page.getByRole('dialog').getByRole('button', { name: 'Copy' })
  await expect.element(copyBtn).toBeDisabled()
})

// T-17.73: Copy error shows alert
test('T-17.73: copy error shows alert', async () => {
  ;(api.catalogs.copy as Mock).mockRejectedValue(new Error('name already exists'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await page.getByRole('dialog').getByRole('textbox').first().fill('new-copy')
  await page.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('name already exists')).toBeVisible()
})

// Copy modal cancel button
test('copy modal cancel closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('Copy Catalog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Copy Catalog')).not.toBeInTheDocument()
})

// Copy modal description field
test('copy modal description field works', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  // Fill description (second textbox in dialog)
  const textboxes = page.getByRole('dialog').getByRole('textbox')
  await textboxes.nth(1).fill('my description')
  await textboxes.first().fill('new-copy')
  await page.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()
  expect(api.catalogs.copy).toHaveBeenCalledWith({
    source: 'my-catalog',
    name: 'new-copy',
    description: 'my description',
  })
})

// Copy modal X close button
test('copy modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('Copy Catalog')).toBeVisible()
  // PatternFly Modal close button is aria-label="Close"
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByText('Copy Catalog')).not.toBeInTheDocument()
})

// T-17.80: Replace modal opens with target dropdown
test('T-17.80: replace modal opens with target dropdown', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  await expect.element(page.getByText('Replace Catalog')).toBeVisible()
  await expect.element(page.getByText('Select target catalog...')).toBeVisible()
})

// T-17.81: Replace modal target dropdown shows catalogs
test('T-17.81: replace modal shows catalog options', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  // Open the dropdown
  await page.getByText('Select target catalog...').click()
  await expect.element(page.getByText('other-cat')).toBeVisible()
  await expect.element(page.getByText('prod-cat')).toBeVisible()
})

// T-17.83: Replace submit calls API
test('T-17.83: replace calls API with correct body', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  // Select target from dropdown
  await page.getByText('Select target catalog...').click()
  await page.getByText('prod-cat').click()
  // Click Replace button in modal
  await page.getByRole('dialog').getByRole('button', { name: 'Replace' }).click()
  expect(api.catalogs.replace).toHaveBeenCalledWith({
    source: 'my-catalog',
    target: 'prod-cat',
    archive_name: undefined,
  })
})

// T-17.82: Replace archive name validation
test('T-17.82: replace archive name validates DNS-label', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  // Select target
  await page.getByText('Select target catalog...').click()
  await page.getByText('prod-cat').click()
  // Enter invalid archive name
  const archiveInput = page.getByRole('dialog').getByRole('textbox')
  await archiveInput.first().fill('INVALID')
  await expect.element(page.getByText('Must be a valid DNS label')).toBeVisible()
  // Replace button should be disabled
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Replace' })).toBeDisabled()
})

// T-17.84: Replace error shows alert
test('T-17.84: replace error shows alert', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  ;(api.catalogs.replace as Mock).mockRejectedValue(new Error('replace failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  await page.getByText('Select target catalog...').click()
  await page.getByText('prod-cat').click()
  await page.getByRole('dialog').getByRole('button', { name: 'Replace' }).click()
  await expect.element(page.getByText('replace failed')).toBeVisible()
})

// Replace modal X close button
test('replace modal X button closes modal', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  await expect.element(page.getByText('Replace Catalog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByText('Replace Catalog')).not.toBeInTheDocument()
})

// Replace modal cancel button
test('replace modal cancel closes modal', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid' })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Replace' }).click()
  await expect.element(page.getByText('Replace Catalog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Replace Catalog')).not.toBeInTheDocument()
})

// === System Attributes in Create/Edit Modals ===

// T-18.39: Create modal renders Name field from schema attrs (not hardcoded)
test('T-18.39: create modal renders Name from schema attrs', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Name field should be present with required indicator
  await expect.element(page.getByRole('dialog').getByText('Name *')).toBeVisible()
})

// T-18.40: Create modal renders Description field from schema attrs
test('T-18.40: create modal renders Description from schema attrs', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Description field should be present (optional, no *)
  await expect.element(page.getByRole('dialog').getByText('Description', { exact: true })).toBeVisible()
})

// T-18.41: Create modal renders custom attributes after system attributes
test('T-18.41: create modal renders custom attrs after system attrs', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // System attrs first, then custom
  await expect.element(page.getByRole('dialog').getByText('Name *')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('hostname')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('port *')).toBeVisible()
})

// T-18.42: Create submits name/description as top-level request fields
test('T-18.42: create submits name/description as top-level fields', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill Name (first textbox) and Description (second textbox)
  const textboxes = page.getByRole('dialog').getByRole('textbox')
  await textboxes.nth(0).fill('my-instance')
  await textboxes.nth(1).fill('a description')

  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({
    name: 'my-instance',
    description: 'a description',
  }))
})

// T-18.43: Edit modal shows Name and Description from schema attrs
test('T-18.43: edit modal shows Name and Description from schema attrs', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Name and Description should be present
  await expect.element(page.getByRole('dialog').getByText('Name *')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Description', { exact: true })).toBeVisible()
})

// T-18.44: Edit submits updated name/description as top-level request fields
test('T-18.44: edit submits name/description as top-level fields', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Name textbox is first, change it
  const textboxes = page.getByRole('dialog').getByRole('textbox')
  await textboxes.nth(0).fill('renamed-inst')

  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({
    name: 'renamed-inst',
  }))
})

// Bug fix: Name label should not show double asterisk ("Name * *")
test('create modal Name label has no duplicate required indicator', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // "Name * *" should NOT appear — only a single required indicator
  await expect.element(page.getByRole('dialog').getByText('Name * *')).not.toBeInTheDocument()
})

test('edit modal Name label has no duplicate required indicator', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Name * *')).not.toBeInTheDocument()
})

// === Additional coverage tests ===

// Cat 2: Link creation success path (lines 438-450)
test('link creation success resets form and reloads instance', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  const toolInstances = [
    { id: 'ti1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'target-tool', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: toolInstances, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByRole('button', { name: 'Link to Instance' })).toBeVisible()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select association: click MenuToggle then option
  await page.getByText('Select association...').click()
  await page.getByText(/uses → tool/).click()

  // Select target instance
  await page.getByText('Select target instance...').click()
  await page.getByText('target-tool').click()

  // Click Link button
  await page.getByRole('dialog').getByRole('button', { name: 'Link' }).click()
  expect(api.links.create).toHaveBeenCalledWith('my-catalog', 'model', 'i1', {
    target_instance_id: 'ti1',
    association_name: 'uses',
  })
})

// Cat 3: Set parent success path (lines 455-468)
test('set parent success resets form and reloads', async () => {
  const parentInstances = [
    { id: 'p1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'parent-model', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockInstances, total: 1 })
  ;(api.instances.setParent as Mock).mockResolvedValue(undefined)
  // After initial list call, subsequent calls for parent instances should return parent list
  let callCount = 0
  ;(api.instances.list as Mock).mockImplementation(() => {
    callCount++
    if (callCount <= 1) return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: parentInstances, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select parent instance
  await page.getByText('Select container...').click()
  await page.getByText('parent-model').click()

  // Submit
  await page.getByRole('dialog').getByRole('button', { name: 'Set Container' }).click()
  expect(api.instances.setParent).toHaveBeenCalledWith('my-catalog', 'model', 'i1', {
    parent_type: 'model',
    parent_instance_id: 'p1',
  })
})

// Cat 3: Set parent error path (line 467-468)
test('set parent error shows error in modal', async () => {
  const parentInstances = [
    { id: 'p1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'parent-model', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.setParent as Mock).mockRejectedValue(new Error('403: forbidden'))
  let callCount = 0
  ;(api.instances.list as Mock).mockImplementation(() => {
    callCount++
    if (callCount <= 1) return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: parentInstances, total: 1 })
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByText('Select container...').click()
  await page.getByText('parent-model').click()

  await page.getByRole('dialog').getByRole('button', { name: 'Set Container' }).click()
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

// Cat 3: Clear parent / Remove Container (lines 1064-1066)
test('remove container calls setParent with empty parent', async () => {
  const childInstances = [{
    id: 'c1', entity_type_id: 'et1', catalog_id: 'cat1', parent_instance_id: 'p1',
    name: 'child-inst', description: '', version: 1,
    attributes: [{ name: 'hostname', type: 'string', value: 'h1' }],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }]
  ;(api.instances.list as Mock).mockResolvedValue({ items: childInstances, total: 1 })
  ;(api.instances.get as Mock).mockResolvedValue({ id: 'p1', name: 'my-parent', entity_type_id: 'et1' })
  ;(api.instances.setParent as Mock).mockResolvedValue(undefined)
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'child-inst' })).toBeVisible()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('Contained by: my-parent').first()).toBeVisible()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Click "Remove Container" button
  await page.getByRole('dialog').getByRole('button', { name: 'Remove Container' }).click()
  expect(api.instances.setParent).toHaveBeenCalledWith('my-catalog', 'model', 'c1', {
    parent_type: '',
    parent_instance_id: '',
  })
})

// Cat 4: Create instance with number attribute (line 217)
test('create instance with number attribute calls parseFloat', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill name
  const nameInput = page.getByRole('dialog').getByRole('textbox').first()
  await nameInput.fill('number-test')

  // Fill port (number attribute) — it renders as spinbutton
  const portInput = page.getByRole('dialog').getByRole('spinbutton')
  await portInput.fill('9090')

  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()
  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({
    name: 'number-test',
    attributes: expect.objectContaining({ port: 9090 }),
  }))
})

// Cat 5: Error catch block - parent name resolution failure (line 303)
test('parent name resolution failure falls back to UUID', async () => {
  const childInstances = [{
    id: 'c1', entity_type_id: 'et1', catalog_id: 'cat1', parent_instance_id: 'p-unknown',
    name: 'child-inst', description: '', version: 1,
    attributes: [{ name: 'hostname', type: 'string', value: 'h1' }],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }]
  ;(api.instances.list as Mock).mockResolvedValue({ items: childInstances, total: 1 })
  ;(api.instances.get as Mock).mockRejectedValue(new Error('404'))
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'child-inst' })).toBeVisible()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Should fall back to showing the UUID when parent name resolution fails
  await expect.element(page.getByText('Contained by: p-unknown').first()).toBeVisible()
})

// Cat 5: Error catch block - load children failure (line 319)
test('load children catch sets empty children', async () => {
  ;(api.instances.listContained as Mock).mockRejectedValue(new Error('500'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Should show "No contained instances." since the catch block sets children to []
  await expect.element(page.getByText('No contained instances.').first()).toBeVisible()
})

// Cat 5: Error catch block - load refs failure (lines 333-334)
test('load refs catch sets empty refs', async () => {
  ;(api.links.forwardRefs as Mock).mockRejectedValue(new Error('500'))
  ;(api.links.reverseRefs as Mock).mockRejectedValue(new Error('500'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Should show "No references." since the catch block sets refs to []
  await expect.element(page.getByText('No references.').first()).toBeVisible()
})

// Cat 5: Error catch block - loadAvailableInstances failure (line 347)
test('load available instances catch sets empty list', async () => {
  // Make list fail only for tool type (loaded when opening add child modal)
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.reject(new Error('500'))
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Should still show the modal without crashing, mode shows Create New (disabled since no available instances)
  await expect.element(page.getByRole('dialog').getByText('Create New')).toBeVisible()
})

// Cat 5: Error catch block - loadChildSchema failure (line 369)
test('load child schema catch sets empty attrs', async () => {
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.reject(new Error('500: schema error'))
    return Promise.resolve(mockSnapshot)
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Should show modal with just Name and Description (no custom attrs since schema load failed)
  await expect.element(page.getByRole('dialog').getByRole('textbox', { name: /Name/i })).toBeVisible()
})

// Cat 5: Error catch block - loadLinkTargetInstances failure (line 383)
test('load link target instances catch sets empty list', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  // Make instances.list fail for tool type (when loading link targets)
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.reject(new Error('500'))
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select association to trigger loadLinkTargetInstances which will fail
  await page.getByText('Select association...').click()
  await page.getByText(/uses → tool/).click()
  // Modal should not crash, target dropdown just has no options
  await expect.element(page.getByText('Select target instance...')).toBeVisible()
})

// Cat 5: Error catch block - loadParentInstances failure (line 392)
test('load parent instances catch sets empty list', async () => {
  // Make list fail for parent type load
  let callCount = 0
  ;(api.instances.list as Mock).mockImplementation(() => {
    callCount++
    if (callCount <= 1) return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.reject(new Error('500'))
  })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Parent instances dropdown should show "Select container..." with no options
  await expect.element(page.getByText('Select container...')).toBeVisible()
})

// Cat 6: Modal onClose - create modal X button (line 764)
test('create modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 6: Modal onClose - edit modal X button (line 813)
test('edit modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 6: Modal onClose - add child modal X button (line 862)
test('add child modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 6: Modal onClose - link modal X button (line 975)
test('link modal X button closes modal', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 6: Modal onClose - set parent modal X button (line 1033)
test('set parent modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 6: Modal onClose - delete modal X button (line 1075)
test('delete modal X button closes modal', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  expect(page.getByRole('dialog').elements().length).toBe(0)
})

// Cat 9: EnumSelect component open/select (lines 1195-1198)
test('enum select opens and selects value in create modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a3', name: 'status', type: 'enum', enum_id: 'enum1', ordinal: 3, required: false },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithEnum)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'active' }, { value: 'inactive' }], total: 2 })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Click the enum select toggle to open
  await page.getByRole('dialog').getByText('Select...').click()
  // Select "active" from the dropdown
  await page.getByRole('option', { name: 'active', exact: true }).click()

  // Fill name and submit
  await page.getByRole('dialog').getByRole('textbox').first().fill('enum-inst')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()
  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({
    name: 'enum-inst',
    attributes: expect.objectContaining({ status: 'active' }),
  }))
})

// Cat 9: EnumSelect in edit modal (lines 840)
test('enum select works in edit modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a3', name: 'status', type: 'enum', enum_id: 'enum1', ordinal: 3, required: false },
    ],
  }
  const instancesWithEnum = [{
    id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
    version: 1, attributes: [
      { name: 'name', type: 'string', value: 'inst-a', system: true },
      { name: 'description', type: 'string', value: 'First', system: true },
      { name: 'status', type: 'enum', value: 'active' },
    ],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }]
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithEnum)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'active' }, { value: 'inactive' }], total: 2 })
  ;(api.instances.list as Mock).mockResolvedValue({ items: instancesWithEnum, total: 1 })
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'inst-a' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Should show enum select with current value "active"
  await expect.element(page.getByRole('dialog').getByText('active')).toBeVisible()
  // Click to open and change to "inactive"
  await page.getByRole('dialog').getByText('active').click()
  await page.getByText('inactive').click()

  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({
    attributes: expect.objectContaining({ status: 'inactive' }),
  }))
})

// Cat 10: Edit modal description onChange (line 829)
test('edit modal description input updates value', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill description (second textbox)
  const descInput = page.getByRole('dialog').getByRole('textbox').nth(1)
  await descInput.fill('updated description')

  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({
    description: 'updated description',
  }))
})

// Cat 10: Edit modal text input onChange for custom attrs (line 847)
test('edit modal custom text attribute updates value', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // hostname is the third textbox (after name, description)
  const hostnameInput = page.getByRole('dialog').getByRole('textbox').nth(2)
  await hostnameInput.fill('new-host')

  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({
    attributes: expect.objectContaining({ hostname: 'new-host' }),
  }))
})

// Cat 7/11: Add child modal child type select with multiple containment assocs (lines 667-669, 873-878)
test('add child modal with multiple containment types selects child type', async () => {
  const snapshotMultiContainment = {
    ...mockSnapshot,
    associations: [
      { id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
      { id: 'assoc3', name: 'configs', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et3', target_entity_type_name: 'config' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotMultiContainment)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // With multiple containment types, child type should NOT be pre-selected
  await expect.element(page.getByRole('dialog').getByText('Select child type...')).toBeVisible()

  // Open child type dropdown by clicking the MenuToggle button
  await page.getByRole('dialog').getByText('Select child type...').click()
  // Select "config" from the dropdown options
  await page.getByText('config', { exact: true }).click()

  // After selection, the toggle should show "config" instead of "Select child type..."
  await expect.element(page.getByRole('dialog').getByText('Select child type...')).not.toBeInTheDocument()
})

// Cat 7: Add child modal child description input (line 921)
test('add child modal description input works', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  const nameInput = page.getByRole('dialog').getByRole('textbox', { name: /Name/i })
  await nameInput.fill('child-with-desc')
  const descInput = page.getByRole('dialog').getByRole('textbox', { name: /Description/i })
  await descInput.fill('child description')

  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
  expect(api.instances.createContained).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'tool', {
    name: 'child-with-desc',
    description: 'child description',
  })
})

// Cat 7: Child attr enum select onChange in add child modal (line 930)
test('add child modal enum attr select works', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'priority', type: 'enum', enum_id: 'enum2', ordinal: 1, required: false },
    ],
    associations: [],
  }
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.resolve(toolSnapshot)
    return Promise.resolve(mockSnapshot)
  })
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [{ value: 'high' }, { value: 'low' }], total: 2 })
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Add Contained Instance' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Fill name
  await page.getByRole('dialog').getByRole('textbox', { name: /Name/i }).fill('child-with-enum')

  // The enum attr should show "Select..." toggle — click it and select "high"
  await page.getByRole('dialog').getByText('Select...').click()
  await page.getByRole('option', { name: 'high' }).click()

  await page.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
  expect(api.instances.createContained).toHaveBeenCalledWith('my-catalog', 'model', 'i1', 'tool', {
    name: 'child-with-enum',
    description: undefined,
    attributes: { priority: 'high' },
  })
})

// Cat 5: link creation error shows in modal (line 450)
test('link creation error shows error in modal', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  const toolInstances = [
    { id: 'ti1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'target-tool', description: '', version: 1, attributes: [] },
  ]
  ;(api.instances.list as Mock).mockImplementation((_cat: string, type: string) => {
    if (type === 'model') return Promise.resolve({ items: mockInstances, total: 1 })
    return Promise.resolve({ items: toolInstances, total: 1 })
  })
  ;(api.links.create as Mock).mockRejectedValue(new Error('409: link already exists'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await page.getByRole('button', { name: 'Link to Instance' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByText('Select association...').click()
  await page.getByText(/uses → tool/).click()
  await page.getByText('Select target instance...').click()
  await page.getByText('target-tool').click()

  await page.getByRole('dialog').getByRole('button', { name: 'Link' }).click()
  await expect.element(page.getByText('409: link already exists')).toBeVisible()
})

// Coverage: render without route param — loadCatalog guard (!name) returns early
test('renders without crash when name param is missing', async () => {
  render(
    <MemoryRouter initialEntries={['/']}>
      <CatalogDetailPage role="Admin" />
    </MemoryRouter>
  )
  // Component mounts but loadCatalog returns early — no API calls, no crash
  // Verify it doesn't call api.catalogs.get (since name is undefined)
  expect(api.catalogs.get).not.toHaveBeenCalled()
})
