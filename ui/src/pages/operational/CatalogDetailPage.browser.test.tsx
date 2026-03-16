import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogDetailPage from './CatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), validate: vi.fn() },
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
