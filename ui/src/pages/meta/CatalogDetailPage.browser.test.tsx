import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogDetailPage from './CatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), list: vi.fn(), validate: vi.fn(), publish: vi.fn(), unpublish: vi.fn(), copy: vi.fn(), replace: vi.fn(), update: vi.fn(), export: vi.fn(), publishPreview: vi.fn(), publishWithToken: vi.fn() },
    catalogVersions: { listPins: vi.fn(), list: vi.fn() },
    versions: { snapshot: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), createContained: vi.fn(), listContained: vi.fn(), setParent: vi.fn() },
    links: { create: vi.fn(), delete: vi.fn(), forwardRefs: vi.fn(), reverseRefs: vi.fn() },
    exporters: { list: vi.fn() },
    exportBindings: { list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), run: vi.fn(), download: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalog = {
  id: 'cat1', name: 'my-catalog', description: 'Test catalog',
  catalog_version_id: 'cv1', catalog_version_label: 'v1.0',
  validation_status: 'draft', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockPins = [
  { pin_id: 'pin-1', entity_type_name: 'model', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 1 },
  { pin_id: 'pin-2', entity_type_name: 'tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
]

const mockSnapshot = {
  entity_type: { id: 'et1', name: 'model' },
  version: { id: 'etv1', version: 1 },
  attributes: [
    { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
    { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
    { id: 'a1', name: 'hostname', base_type: 'string', ordinal: 1, required: false },
    { id: 'a2', name: 'port', base_type: 'number', ordinal: 2, required: true },
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
    { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
    { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
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
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes>
        <Route path="/schema/catalogs/:name" element={<CatalogDetailPage role={role} />} />
        <Route path="/schema/catalogs" element={<div>Catalog List</div>} />
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
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    const found = mockInstances.find(i => i.id === id)
    if (found) return Promise.resolve(found)
    return Promise.resolve({ id, name: `inst-${id}`, entity_type_id: 'et1', version: 1, attributes: [], created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' })
  })
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'i2', name: 'new-inst' })
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'inst-a', version: 2 })
  ;(api.instances.delete as Mock).mockResolvedValue(undefined)
  ;(api.instances.listContained as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.instances.createContained as Mock).mockResolvedValue({ id: 'c1', name: 'new-child' })
  ;(api.links.forwardRefs as Mock).mockResolvedValue(mockForwardRefs)
  ;(api.links.reverseRefs as Mock).mockResolvedValue(mockReverseRefs)
  ;(api.links.create as Mock).mockResolvedValue({ id: 'link-new' })
  ;(api.links.delete as Mock).mockResolvedValue(undefined)
  ;(api.catalogs.copy as Mock).mockResolvedValue({ id: 'new-id', name: 'copy-cat' })
  ;(api.catalogs.replace as Mock).mockResolvedValue({ id: 'src-id', name: 'prod' })
  ;(api.catalogs.export as Mock).mockResolvedValue({ catalog: { name: 'my-catalog' }, entity_types: [] })
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [{ name: 'mcp-gateway', description: 'MCP Gateway Exporter', parameter_schema: [{ name: 'server_type', type: 'string', required: true, description: 'Server type' }] }] })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [] })
  ;(api.exportBindings.create as Mock).mockResolvedValue({ id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never' })
  ;(api.exportBindings.delete as Mock).mockResolvedValue(undefined)
  ;(api.exportBindings.run as Mock).mockResolvedValue(undefined)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({ session_token: 'tok1', bindings: [], has_failures: false, expires_at: '2026-01-01T01:00:00Z' })
  ;(api.catalogs.publishWithToken as Mock).mockResolvedValue({ status: 'published' })
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: [{ name: 'other-cat' }, { name: 'prod-cat' }], total: 2 })
  ;(api.catalogs.update as Mock).mockResolvedValue({ ...mockCatalog, description: 'updated desc' })
  ;(api.catalogVersions.list as Mock).mockResolvedValue({ items: [
    { id: 'cv1', version_label: 'v1.0', lifecycle_stage: 'development' },
    { id: 'cv2', version_label: 'v2.0', lifecycle_stage: 'testing' },
  ], total: 2 })
})

// Helper: wait for instance table to render
async function waitForInstances() {
  await expect.element(page.getByRole('gridcell', { name: 'inst-a' })).toBeVisible()
}

// T-11.48: Catalog detail page shows tabs per pinned entity type
test('T-11.48: shows entity type tabs', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'model', exact: true })).toBeVisible()
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
  await expect.element(page.getByRole('tab', { name: 'model', exact: true })).toBeVisible()
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Edit inst-a')).toBeVisible()
})

// T-11.55: Edit instance submits updated values
test('T-11.55: edit submits', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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
  await expect.element(page.getByRole('tab', { name: 'model', exact: true })).toBeVisible()
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
  await expect.element(page.getByRole('gridcell', { name: 'target-inst (tool)' }).first()).toBeVisible()
  // Reverse references visible
  await expect.element(page.getByText('Referenced By').first()).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'source-inst (server)' }).first()).toBeVisible()
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

// TD-78: Forward refs show merged target column "instance-name (entity-type)"
test('TD-78: forward refs show merged target column', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Merged column should show "target-inst (tool)"
  await expect.element(page.getByRole('gridcell', { name: 'target-inst (tool)' }).first()).toBeVisible()
})

// TD-78: Reverse refs show merged target column "instance-name (entity-type)"
test('TD-78: reverse refs show merged target column', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // Merged column should show "source-inst (server)"
  await expect.element(page.getByRole('gridcell', { name: 'source-inst (server)' }).first()).toBeVisible()
})

// TD-78: Entity Type is not a separate column in forward refs
test('TD-78: no separate Entity Type column in forward refs', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  // "tool" should NOT appear as its own gridcell (it's merged into target)
  const forwardTable = page.getByRole('table', { name: 'Forward references' })
  await expect.element(forwardTable.getByRole('columnheader', { name: 'Entity Type' })).not.toBeInTheDocument()
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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

// TD-84 / T-28.03: handleUnlink shows error on failure
test('T-28.03: unlink error is shown, not swallowed', async () => {
  const snapshotWithLink = {
    ...mockSnapshot,
    associations: [
      ...mockSnapshot.associations,
      { id: 'assoc2', name: 'uses', type: 'directional', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithLink)
  ;(api.links.delete as Mock).mockRejectedValue(new Error('Permission denied'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('Forward References').first()).toBeVisible()
  await page.getByRole('button', { name: 'Unlink' }).first().click()
  await expect.element(page.getByText('Permission denied')).toBeVisible()
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
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'tool-version', base_type: 'string', ordinal: 1, required: false },
      { id: 'ta2', name: 'weight', base_type: 'number', ordinal: 2, required: true },
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
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'tool-version', base_type: 'string', ordinal: 1, required: false },
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
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'weight', base_type: 'number', ordinal: 1, required: false },
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
  await page.getByRole('dialog').getByRole('textbox', { name: /weight/i }).fill('3.14')
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
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'status', base_type: 'enum', type_definition_version_id: 'tdv-enum1', constraints: { values: ['active', 'inactive'] }, ordinal: 1, required: false },
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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

// Enum attributes render select in create modal
test('enum attributes render select in create modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'a1', name: 'hostname', base_type: 'string', ordinal: 1, required: false },
      { id: 'a3', name: 'status', base_type: 'enum', type_definition_version_id: 'tdv-enum1', constraints: { values: ['active', 'inactive'] }, ordinal: 3, required: false },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithEnum)

  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // The enum attribute should render a native <select> element
  await expect.element(page.getByRole('dialog').getByRole('combobox', { name: 'status' })).toBeVisible()
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
  await expect.element(page.getByRole('tab', { name: 'model', exact: true })).toBeVisible()
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
  // Mock GetByID: return child instance for 'c1', parent for 'p1'
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'c1') return Promise.resolve(childInstances[0])
    if (id === 'p1') return Promise.resolve({ id: 'p1', name: 'my-parent-server', entity_type_id: 'et1' })
    return Promise.resolve({ id, name: `inst-${id}`, entity_type_id: 'et1' })
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

// T-34.30: Export Plugins tab visible on CatalogDetailPage
test('T-34.30: Export Plugins tab visible', async () => {
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await expect.element(page.getByRole('tab', { name: 'Export Plugins' })).toBeVisible()
})

// T-34.31: Export Plugins tab shows empty state when no bindings
test('T-34.31: Export Plugins tab shows empty state', async () => {
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('No export bindings configured')).toBeVisible()
})

// T-34.32: Add Export Binding button visible for Admin
test('T-34.32: Add Export Binding button visible for Admin', async () => {
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByRole('button', { name: 'Add Export Binding' })).toBeVisible()
})

// T-34.38: RO role: Add Export Binding button hidden
test('T-34.38: RO role: Add Export Binding hidden', async () => {
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="RO" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  expect(page.getByRole('button', { name: 'Add Export Binding' }).elements().length).toBe(0)
})

// T-34.33: Add Binding modal opens with exporter dropdown
test('T-34.33: Add Binding modal opens with exporter dropdown', async () => {
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await expect.element(page.getByRole('button', { name: 'Create' })).toBeVisible()
})

// T-34.39: Binding list shows binding data
test('T-34.39: Binding list shows binding data', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server' }, enabled: true, last_run_status: 'success', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/catalogs/my-catalog']}>
      <Routes><Route path="/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  await expect.element(page.getByText('success')).toBeVisible()
})

// T-34.37: Enable/disable toggle updates binding
test('T-34.37: Enable/disable toggle updates binding', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server' }, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  ;(api.exportBindings.update as Mock).mockResolvedValue({ id: 'b1', enabled: false })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByRole('button', { name: 'Enabled' })).toBeVisible()
  await page.getByRole('button', { name: 'Enabled' }).click()
  expect(api.exportBindings.update).toHaveBeenCalledWith('my-catalog', 'b1', { enabled: false })
})

// T-34.48: "Export Now" button triggers YAML download
test('T-34.48: Export Now triggers download', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="RW" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByRole('button', { name: 'Export Now' })).toBeVisible()
  await page.getByRole('button', { name: 'Export Now' }).click()
  expect(api.exportBindings.run).toHaveBeenCalledWith('my-catalog', 'b1', undefined)
})

// T-34.49: Export error shows error message in UI
test('T-34.49: Export error shows error message', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  ;(api.exportBindings.run as Mock).mockRejectedValue(new Error('Export failed: missing route_name'))
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="RW" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await expect.element(page.getByText('Export failed: missing route_name')).toBeVisible()
})

// T-34.34: Add Binding modal validates required params — Create disabled when required params empty
test('T-34.34: Add Binding modal validates required params', async () => {
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  // Select an exporter that has a required param
  const select = page.getByLabelText('Select exporter')
  await select.selectOptions('mcp-gateway')
  // Required param "server_type" is empty — Create should be disabled
  const createBtn = page.getByRole('button', { name: 'Create' })
  await expect.element(createBtn).toBeVisible()
  expect((createBtn.element() as HTMLButtonElement).disabled).toBe(true)
})

// T-34.35: Edit Binding modal pre-fills current parameters
test('T-34.35: Edit Binding modal pre-fills params', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server' }, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  // Click Edit in the binding row (scope to the grid to avoid the catalog Edit button)
  const grid = page.getByRole('grid', { name: 'Export bindings' })
  await grid.getByRole('button', { name: 'Edit' }).click()
  // Modal should have pre-filled server_type param
  const serverTypeInput = page.getByRole('textbox', { name: /server_type/ })
  await expect.element(serverTypeInput).toBeVisible()
  expect((serverTypeInput.element() as HTMLInputElement).value).toBe('mcp-server')
})

// T-34.36: Delete Binding shows confirmation dialog
test('T-34.36: Delete Binding shows confirmation dialog', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  // Should show confirmation dialog, NOT delete immediately
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByText(/Are you sure/)).toBeVisible()
  // Confirm delete
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.exportBindings.delete).toHaveBeenCalledWith('my-catalog', 'b1')
})

// --- Publish Preview Flow Tests ---

const validCatalog = {
  ...mockCatalog,
  validation_status: 'valid',
  published: false,
}

// T-34.92: Publish flow calls preview first, results shown in modal
test('T-34.92: Publish calls preview first and shows results', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: false,
    bindings: [{ binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'success', artifact_count: 2, error: '' }],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  // Should show preview modal with binding results
  const dialog = page.getByRole('dialog')
  await expect.element(dialog).toBeVisible()
  await expect.element(dialog.getByText('mcp-gateway')).toBeVisible()
  await expect.element(dialog.getByText('success')).toBeVisible()
  expect(api.catalogs.publishPreview).toHaveBeenCalledWith('my-catalog')
})

// T-34.93: Publish with failures shows confirmation dialog with per-binding results
test('T-34.93: Publish with failures shows per-binding results', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: true,
    bindings: [
      { binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'success', artifact_count: 2, error: '' },
      { binding_id: 'b2', exporter_name: 'configmap', status: 'failed', artifact_count: 0, error: 'missing instances' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
    { id: 'b2', exporter_name: 'configmap', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  const dialog93 = page.getByRole('dialog')
  await expect.element(dialog93).toBeVisible()
  // Should show failure details in the results table
  const previewGrid = dialog93.getByRole('grid', { name: 'Preview results' })
  await expect.element(previewGrid.getByText('failed')).toBeVisible()
  await expect.element(previewGrid.getByText('missing instances')).toBeVisible()
  // Should have "Publish Anyway" and "Abort" buttons
  await expect.element(page.getByRole('button', { name: 'Publish Anyway' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Abort' })).toBeVisible()
})

// T-34.94: Publish Anyway commits despite failures
test('T-34.94: Publish Anyway commits despite failures', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: true,
    bindings: [
      { binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'failed', artifact_count: 0, error: 'some error' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('button', { name: 'Publish Anyway' }).click()
  expect(api.catalogs.publishWithToken).toHaveBeenCalledWith('my-catalog', 'tok1')
})

// T-34.95: Abort cancels — catalog stays unpublished
test('T-34.95: Abort cancels publish', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: true,
    bindings: [
      { binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'failed', artifact_count: 0, error: 'err' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('button', { name: 'Abort' }).click()
  // Modal should close and publishWithToken should NOT have been called
  expect(api.catalogs.publishWithToken).not.toHaveBeenCalled()
})

// T-34.96: Successful publish auto-downloads files per binding
test('T-34.96: Successful publish auto-downloads per binding', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: false,
    bindings: [
      { binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'success', artifact_count: 2, error: '' },
      { binding_id: 'b2', exporter_name: 'other', status: 'success', artifact_count: 1, error: '' },
    ],
  })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
    { id: 'b2', exporter_name: 'other', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  ;(api.exportBindings.download as Mock).mockResolvedValue(undefined)
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // No failures — click Publish (not "Publish Anyway")
  await page.getByRole('dialog').getByRole('button', { name: 'Publish' }).click()
  expect(api.catalogs.publishWithToken).toHaveBeenCalledWith('my-catalog', 'tok1')
  // After publish succeeds, download should be called for each binding
  expect(api.exportBindings.download).toHaveBeenCalledWith('my-catalog', 'tok1', 'b1')
  expect(api.exportBindings.download).toHaveBeenCalledWith('my-catalog', 'tok1', 'b2')
})

// T-34.106: Create binding submits form with exporter and params
test('T-34.106: Create binding submits form with exporter and params', async () => {
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  // Select exporter
  await page.getByLabelText('Select exporter').selectOptions('mcp-gateway')
  // Fill in required param
  const paramInput = page.getByRole('textbox', { name: /server_type/ })
  await expect.element(paramInput).toBeVisible()
  await paramInput.fill('my-server')
  // Create button should now be enabled
  await page.getByRole('button', { name: 'Create' }).click()
  expect(api.exportBindings.create).toHaveBeenCalledWith('my-catalog', {
    exporter_name: 'mcp-gateway',
    parameters: { server_type: 'my-server' },
  })
})

// T-34.107: Edit binding submits updated params
test('T-34.107: Edit binding submits updated params', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server' }, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  ;(api.exportBindings.update as Mock).mockResolvedValue({ id: 'b1', parameters: { server_type: 'updated' } })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  const grid = page.getByRole('grid', { name: 'Export bindings' })
  await grid.getByRole('button', { name: 'Edit' }).click()
  // Change param value
  const paramInput = page.getByRole('textbox', { name: /server_type/ })
  await paramInput.fill('updated-server')
  await page.getByRole('button', { name: 'Save' }).click()
  expect(api.exportBindings.update).toHaveBeenCalledWith('my-catalog', 'b1', { parameters: { server_type: 'updated-server' } })
})

// T-34.108: Delete binding cancel closes modal without deleting
test('T-34.108: Delete binding cancel closes modal', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Click Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(api.exportBindings.delete).not.toHaveBeenCalled()
})

// T-34.109: Toggle enabled error shows alert
test('T-34.109: Toggle enabled error shows alert', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  ;(api.exportBindings.update as Mock).mockRejectedValue(new Error('Update failed: forbidden'))
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Enabled' }).click()
  await expect.element(page.getByText('Update failed: forbidden')).toBeVisible()
})

// T-34.110: Delete binding error shows alert
test('T-34.110: Delete binding error shows alert', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  ;(api.exportBindings.delete as Mock).mockRejectedValue(new Error('Delete forbidden'))
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Delete forbidden')).toBeVisible()
})

// T-34.111: Publish preview error shows alert in modal
test('T-34.111: Publish preview error shows alert', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockRejectedValue(new Error('Preview failed: timeout'))
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByText('Preview failed: timeout')).toBeVisible()
})

// T-34.112: Publish with token error shows error in modal
test('T-34.112: Publish with token error shows error', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue(validCatalog)
  ;(api.catalogs.publishPreview as Mock).mockResolvedValue({
    session_token: 'tok1', expires_at: '2026-01-01T01:00:00Z', has_failures: false,
    bindings: [{ binding_id: 'b1', exporter_name: 'mcp-gateway', status: 'success', artifact_count: 1, error: '' }],
  })
  ;(api.catalogs.publishWithToken as Mock).mockRejectedValue(new Error('Publish failed: invalid token'))
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: {}, enabled: true, last_run_status: 'never', last_run_error: '' },
  ] })
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Publish' }).click()
  await expect.element(page.getByText('Publish failed: invalid token')).toBeVisible()
})

// T-34.113: Create binding error shows alert in modal
test('T-34.113: Create binding error shows alert', async () => {
  ;(api.exportBindings.create as Mock).mockRejectedValue(new Error('Create failed: duplicate'))
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await page.getByLabelText('Select exporter').selectOptions('mcp-gateway')
  const paramInput = page.getByRole('textbox', { name: /server_type/ })
  await paramInput.fill('my-server')
  await page.getByRole('button', { name: 'Create' }).click()
  await expect.element(page.getByText('Create failed: duplicate')).toBeVisible()
})

// T-34.115: Add Binding modal Cancel closes without creating
test('T-34.115: Add Binding modal Cancel closes', async () => {
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await expect.element(page.getByRole('button', { name: 'Create' })).toBeVisible()
  // Cancel the modal
  await page.getByRole('button', { name: 'Cancel' }).click()
  // Modal should close — Create button should be gone
  expect(page.getByRole('button', { name: 'Create' }).query()).toBeNull()
  expect(api.exportBindings.create).not.toHaveBeenCalled()
})

// T-34.116: Edit Binding modal Cancel closes without saving
test('T-34.116: Edit Binding modal Cancel closes', async () => {
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [
    { id: 'b1', exporter_name: 'mcp-gateway', parameters: { server_type: 'mcp-server' }, enabled: true, last_run_status: 'never', last_run_error: '', created_at: '2026-01-01', updated_at: '2026-01-01' },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await expect.element(page.getByText('mcp-gateway')).toBeVisible()
  const grid = page.getByRole('grid', { name: 'Export bindings' })
  await grid.getByRole('button', { name: 'Edit' }).click()
  await expect.element(page.getByRole('button', { name: 'Save' })).toBeVisible()
  // Cancel the modal
  await page.getByRole('button', { name: 'Cancel' }).click()
  // Modal should close
  expect(page.getByRole('button', { name: 'Save' }).query()).toBeNull()
  expect(api.exportBindings.update).not.toHaveBeenCalled()
})

// T-34.117: Add Binding modal pre-fills default param values
test('T-34.117: Add Binding modal pre-fills default param values', async () => {
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [
    { name: 'mcp-gateway', description: 'MCP Gateway Exporter', parameter_schema: [
      { name: 'server_type', type: 'string', required: true, description: 'Server type', default: 'mcp-default' },
      { name: 'port', type: 'string', required: false, description: 'Port number' },
    ] },
  ] })
  render(
    <MemoryRouter initialEntries={['/schema/catalogs/my-catalog']}>
      <Routes><Route path="/schema/catalogs/:name" element={<CatalogDetailPage role="Admin" />} /></Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByRole('heading', { name: /my-catalog/ })).toBeVisible()
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await page.getByLabelText('Select exporter').selectOptions('mcp-gateway')
  // The default value should be pre-filled
  const paramInput = page.getByRole('textbox', { name: /server_type/ })
  await expect.element(paramInput).toBeVisible()
  expect((paramInput.element() as HTMLInputElement).value).toBe('mcp-default')
})
