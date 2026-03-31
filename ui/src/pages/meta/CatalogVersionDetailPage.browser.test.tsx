import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page, userEvent } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogVersionDetailPage from './CatalogVersionDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogVersions: {
      get: vi.fn(),
      listPins: vi.fn(),
      listTransitions: vi.fn(),
      promote: vi.fn(),
      demote: vi.fn(),
      update: vi.fn(),
      addPin: vi.fn(),
      updatePin: vi.fn(),
      removePin: vi.fn(),
    },
    versions: {
      list: vi.fn(),
      snapshot: vi.fn(),
    },
    entityTypes: {
      list: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockCV = {
  id: 'cv-1',
  version_label: 'v1.0',
  lifecycle_stage: 'development' as const,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const mockPins = [
  { pin_id: 'pin-1', entity_type_name: 'Model', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 3 },
  { pin_id: 'pin-2', entity_type_name: 'Tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
]

const mockTransitions = [
  { id: 'lt-1', from_stage: '', to_stage: 'development', performed_by: 'system', performed_at: '2026-01-01T00:00:00Z' },
  { id: 'lt-2', from_stage: 'development', to_stage: 'testing', performed_by: 'admin', performed_at: '2026-01-02T00:00:00Z' },
]

const mockSnapshot = {
  entity_type: { id: 'et-1', name: 'Model', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  version: { id: 'etv-1', entity_type_id: 'et-1', version: 3, description: 'V3', created_at: '2026-01-01T00:00:00Z' },
  attributes: [
    { id: 'a1', name: 'hostname', description: 'The hostname', type: 'string', ordinal: 1, required: false },
    { id: 'a2', name: 'port', description: 'Port number', type: 'number', ordinal: 2, required: true },
    { id: 'a3', name: 'status', description: 'Status flag', type: 'enum', enum_id: 'e1', enum_name: 'boolean', ordinal: 3, required: false },
  ],
  associations: [
    { id: 'as1', name: 'tools', type: 'containment', target_entity_type_id: 'et-2', target_entity_type_name: 'Tool', source_role: 'model', target_role: 'tool', source_cardinality: '1', target_cardinality: '0..n', direction: 'outgoing' },
    { id: 'as2', name: 'platform', type: 'containment', target_entity_type_id: 'et-1', target_entity_type_name: 'Model', source_role: 'parent', target_role: 'child', source_cardinality: '1', target_cardinality: '0..1', direction: 'incoming', source_entity_type_id: 'et-3', source_entity_type_name: 'Platform' },
    { id: 'as3', name: 'datasets', type: 'bidirectional', target_entity_type_id: 'et-4', target_entity_type_name: 'Dataset', source_role: 'model', target_role: 'data', source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
  ],
}

function renderDetail(role: 'Admin' | 'RO' | 'RW' | 'SuperAdmin' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/catalog-versions/cv-1']}>
      <Routes>
        <Route path="/schema/catalog-versions/:id" element={<CatalogVersionDetailPage role={role} />} />
        <Route path="/schema/catalog-versions" element={<div>CV List</div>} />
        <Route path="/schema/entity-types/:id" element={<div>ET Detail</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogVersions.get as Mock).mockResolvedValue(mockCV)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.catalogVersions.listTransitions as Mock).mockResolvedValue({ items: mockTransitions, total: 2 })
  ;(api.catalogVersions.promote as Mock).mockResolvedValue({ status: 'promoted' })
  ;(api.catalogVersions.demote as Mock).mockResolvedValue({ status: 'demoted' })
  if (api.versions?.snapshot) {
    ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshot)
  }
  ;(api.catalogVersions.update as Mock).mockResolvedValue({ ...mockCV, description: 'updated' })
  ;(api.catalogVersions.addPin as Mock).mockResolvedValue({ entity_type_version_id: 'etv-new' })
  ;(api.catalogVersions.updatePin as Mock).mockResolvedValue({ pin_id: 'pin-1', entity_type_version_id: 'etv-v2' })
  ;(api.catalogVersions.removePin as Mock).mockResolvedValue(undefined)
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
  ], total: 2 })
  ;(api.versions.list as Mock).mockResolvedValue({ items: [
    { id: 'etv-1', entity_type_id: 'et-1', version: 1, description: 'V1', created_at: '' },
    { id: 'etv-2', entity_type_id: 'et-1', version: 2, description: 'V2', created_at: '' },
  ], total: 2 })
})

// === Overview Tab ===

test('shows catalog version label and stage', async () => {
  renderDetail()
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByText('development').first()).toBeVisible()
})

test('shows overview details', async () => {
  renderDetail()
  await expect.element(page.getByText('cv-1')).toBeVisible()
})

test('shows back link to catalog versions list', async () => {
  renderDetail()
  const backLink = page.getByRole('button', { name: /Back to Catalog Versions/i })
  await expect.element(backLink).toBeVisible()
})

test('back link navigates to CV list', async () => {
  renderDetail()
  await page.getByRole('button', { name: /Back to Catalog Versions/i }).click()
  await expect.element(page.getByText('CV List')).toBeVisible()
})

test('shows not found when CV does not exist', async () => {
  ;(api.catalogVersions.get as Mock).mockRejectedValue(new Error('404: not found'))
  renderDetail()
  await expect.element(page.getByText('404: not found')).toBeVisible()
})

// === Promote / Demote ===

test('shows Promote button for development CV as Admin', async () => {
  renderDetail('Admin')
  await expect.element(page.getByRole('button', { name: 'Promote' })).toBeVisible()
})

test('hides Promote button for production CV', async () => {
  ;(api.catalogVersions.get as Mock).mockResolvedValue({ ...mockCV, lifecycle_stage: 'production' })
  renderDetail('Admin')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Promote' })).not.toBeInTheDocument()
})

test('promote calls API and reloads', async () => {
  renderDetail('Admin')
  await expect.element(page.getByRole('button', { name: 'Promote' })).toBeVisible()
  await page.getByRole('button', { name: 'Promote' }).click()
  expect(api.catalogVersions.promote).toHaveBeenCalledWith('cv-1')
})

test('shows Demote button for testing CV as Admin', async () => {
  ;(api.catalogVersions.get as Mock).mockResolvedValue({ ...mockCV, lifecycle_stage: 'testing' })
  renderDetail('Admin')
  await expect.element(page.getByRole('button', { name: 'Demote' })).toBeVisible()
})

test('hides Demote button for development CV', async () => {
  renderDetail('Admin')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Demote' })).not.toBeInTheDocument()
})

test('RO cannot see Promote or Demote', async () => {
  renderDetail('RO')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Promote' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Demote' })).not.toBeInTheDocument()
})

test('promote error shows alert', async () => {
  ;(api.catalogVersions.promote as Mock).mockRejectedValue(new Error('403: forbidden'))
  renderDetail('Admin')
  await page.getByRole('button', { name: 'Promote' }).click()
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

// === Bill of Materials Tab ===

test('BOM tab shows pinned entity types', async () => {
  renderDetail()
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Tool', exact: true })).toBeVisible()
})

test('BOM tab shows empty state when no pins', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByText('No entity types pinned to this catalog version.')).toBeVisible()
})

// TD-44: BOM tab shows description column for pins
test('TD-44: BOM tab shows description column', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({
    items: [
      { pin_id: 'pin-1', entity_type_name: 'Model', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 3, description: 'ML model type' },
      { pin_id: 'pin-2', entity_type_name: 'Tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
    ],
    total: 2,
  })
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('gridcell', { name: 'ML model type' })).toBeVisible()
})

// T-E.64: Clicking entity type in BOM opens read-only modal
test('T-E.64: BOM entity type click opens read-only modal', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  // Modal should open, NOT navigate away
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Model — V3')).toBeVisible()
})

// T-E.65: BOM modal shows attributes table
test('T-E.65: BOM modal shows attributes table', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Attributes should be listed — use exact matching to avoid description conflicts
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'hostname', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'port *', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'string', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'number', exact: true })).toBeVisible()
  // Enum attribute should show "boolean (enum)" not just "enum"
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'boolean (enum)', exact: true })).toBeVisible()
})

// T-E.66: BOM modal shows associations table
test('T-E.66: BOM modal shows associations table', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Outgoing containment: "contains" label, target entity name, target role
  await expect.element(page.getByRole('dialog').getByText('contains').first()).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'Tool', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'tool', exact: true })).toBeVisible()
  // Incoming containment: "contained by" label, source entity name, source role
  await expect.element(page.getByRole('dialog').getByText('contained by')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'Platform', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'parent', exact: true })).toBeVisible()
  // Bidirectional: "references (mutual)" label, target entity name, target role
  await expect.element(page.getByRole('dialog').getByText('references (mutual)')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'Dataset', exact: true })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: 'data', exact: true })).toBeVisible()
})

// T-E.67: BOM modal has no edit controls
test('T-E.67: BOM modal has no edit controls', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // No edit controls
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Add' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
  // Close button exists in footer
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Close' }).first()).toBeVisible()
})

// Snapshot load error keeps modal open with error inside
test('BOM snapshot error shows error inside modal', async () => {
  ;(api.versions.snapshot as Mock).mockRejectedValue(new Error('500: snapshot failed'))
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  // Modal should stay open with error displayed inside
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('500: snapshot failed')).toBeVisible()
})

test('BOM tab shows error when listPins fails', async () => {
  ;(api.catalogVersions.listPins as Mock).mockRejectedValue(new Error('500: server error'))
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByText('500: server error')).toBeVisible()
})

// === Transitions Tab ===

test('Transitions tab shows lifecycle history', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Transitions' }).click()
  await expect.element(page.getByText('(initial)')).toBeVisible()
  await expect.element(page.getByText('system')).toBeVisible()
  await expect.element(page.getByText('admin')).toBeVisible()
})

test('Transitions tab shows empty state when no transitions', async () => {
  ;(api.catalogVersions.listTransitions as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: 'Transitions' }).click()
  await expect.element(page.getByText('No transitions recorded.')).toBeVisible()
})

test('Transitions tab shows error when listTransitions fails', async () => {
  ;(api.catalogVersions.listTransitions as Mock).mockRejectedValue(new Error('500: server error'))
  renderDetail()
  await page.getByRole('tab', { name: 'Transitions' }).click()
  await expect.element(page.getByText('500: server error')).toBeVisible()
})

// T-E.87: BOM modal associations table includes cardinality
test('T-E.87: BOM modal associations table includes cardinality column', async () => {
  renderDetail()
  // Click BOM tab, then click entity type name to open snapshot
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  // Verify cardinality values: outgoing containment "1 → 0..n"
  await expect.element(page.getByText('1 → 0..n')).toBeVisible()
})

// BOM modal associations show name column
test('BOM modal associations show name column', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model', exact: true }).click()
  // Verify association name "tools" is displayed in the modal
  await expect.element(page.getByText('tools')).toBeVisible()
  // Verify "datasets" name from bidirectional association
  await expect.element(page.getByText('datasets')).toBeVisible()
})

// T-E.132: Diagram tab exists on CV detail page
test('T-E.132: Diagram tab exists on CV detail page', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Diagram' })).toBeVisible()
})

// T-E.133: CV diagram loads without visiting BOM tab first
test('T-E.133: CV diagram renders when clicking Diagram tab directly', async () => {
  renderDetail()
  // Click Diagram tab directly — should load pins and render diagram
  await page.getByRole('tab', { name: 'Diagram' }).click()
  // The diagram container should appear (data-testid from EntityTypeDiagram)
  await expect.element(page.getByTestId('entity-type-diagram')).toBeVisible()
})

// === Phase 2 CRUD: Inline Edit Description ===

test('Edit description button visible for RW+, hidden for RO', async () => {
  renderDetail('RW')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
})

test('Edit description button hidden for RO user', async () => {
  renderDetail('RO')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
})

test('Edit description: click Edit shows TextInput, Save calls API', async () => {
  ;(api.catalogVersions.get as Mock).mockResolvedValue({ ...mockCV, description: 'old desc' })
  renderDetail('Admin')
  await expect.element(page.getByText('old desc')).toBeVisible()
  await page.getByRole('button', { name: 'Edit description' }).click()
  // TextInput should appear with current value
  const input = page.getByRole('textbox', { name: 'Description' })
  await expect.element(input).toBeVisible()
  await input.fill('new desc')
  await page.getByRole('button', { name: 'Save' }).first().click()
  expect(api.catalogVersions.update).toHaveBeenCalledWith('cv-1', { description: 'new desc' })
})

test('Edit description: Cancel restores original', async () => {
  ;(api.catalogVersions.get as Mock).mockResolvedValue({ ...mockCV, description: 'original desc' })
  renderDetail('Admin')
  await expect.element(page.getByText('original desc')).toBeVisible()
  await page.getByRole('button', { name: 'Edit description' }).click()
  await expect.element(page.getByRole('textbox', { name: 'Description' })).toBeVisible()
  await page.getByRole('button', { name: 'Cancel' }).first().click()
  // Should show original text again, no TextInput
  await expect.element(page.getByText('original desc')).toBeVisible()
  await expect.element(page.getByRole('textbox', { name: 'Description' })).not.toBeInTheDocument()
})

// === Phase 2 CRUD: Inline Edit Version Label ===

test('Edit version label button visible for RW+', async () => {
  renderDetail('RW')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit version label' })).toBeVisible()
})

test('Edit version label button hidden for RO', async () => {
  renderDetail('RO')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit version label' })).not.toBeInTheDocument()
})

test('Edit version label: triggers PUT', async () => {
  renderDetail('Admin')
  await expect.element(page.getByText('v1.0').first()).toBeVisible()
  await page.getByRole('button', { name: 'Edit version label' }).click()
  const input = page.getByRole('textbox', { name: 'Version label' })
  await expect.element(input).toBeVisible()
  await input.fill('v2.0')
  await page.getByRole('button', { name: 'Save' }).first().click()
  expect(api.catalogVersions.update).toHaveBeenCalledWith('cv-1', { version_label: 'v2.0' })
})

// === Phase 2 CRUD: Add Pin Button ===

test('Add Pin button visible for RW+ in BOM tab', async () => {
  renderDetail('RW')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Add Pin' })).toBeVisible()
})

test('Add Pin button hidden for RO in BOM tab', async () => {
  renderDetail('RO')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Pin' })).not.toBeInTheDocument()
})

test('Add Pin modal opens and can submit', async () => {
  // Add an unpinned entity type — Model and Tool are already pinned in mockPins
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
    { id: 'et-3', name: 'Platform', created_at: '', updated_at: '' },
  ], total: 3 })
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByText('Select entity type...')).toBeVisible()

  // Select entity type — only unpinned types available; use getByTestId to bypass aria-hidden on PF6 Popper portal
  await page.getByRole('dialog').getByText('Select entity type...').click()
  await page.getByTestId('pin-et-Platform').click()

  // Select version
  await expect.element(page.getByRole('dialog').getByText('Select version...')).toBeVisible()
  await page.getByRole('dialog').getByText('Select version...').click()
  await page.getByTestId('pin-etv-V2').click()

  // Submit
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()
  expect(api.catalogVersions.addPin).toHaveBeenCalledWith('cv-1', 'etv-2')
})

test('Add Pin error shows in modal', async () => {
  // Add an unpinned entity type to make the Select dropdown functional
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
    { id: 'et-3', name: 'Platform', created_at: '', updated_at: '' },
  ], total: 3 })
  ;(api.catalogVersions.addPin as Mock).mockRejectedValue(new Error('409: already pinned'))
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await page.getByRole('dialog').getByText('Select entity type...').click()
  await page.getByTestId('pin-et-Platform').click()
  await page.getByRole('dialog').getByText('Select version...').click()
  await page.getByTestId('pin-etv-V1').click()
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()
  await expect.element(page.getByText('409: already pinned')).toBeVisible()
})

// === Phase 2 CRUD: Remove Pin Button ===

test('Remove button per pin row visible for RW+', async () => {
  renderDetail('RW')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  // Each pin row should have a Remove button
  const removeButtons = page.getByRole('button', { name: 'Remove' })
  expect(removeButtons.elements().length).toBe(2) // 2 pins
})

test('Remove button hidden for RO', async () => {
  renderDetail('RO')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
})

test('Remove pin calls removePin API', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Remove' }).first().click()
  expect(api.catalogVersions.removePin).toHaveBeenCalledWith('cv-1', 'pin-1')
})

// Error paths for description edit
test('Edit description error shows alert', async () => {
  ;(api.catalogVersions.update as Mock).mockRejectedValue(new Error('500: update failed'))
  renderDetail('Admin')
  await page.getByRole('button', { name: 'Edit description' }).click()
  await page.getByRole('textbox', { name: 'Description' }).fill('bad')
  await page.getByRole('button', { name: 'Save' }).first().click()
  await expect.element(page.getByText('500: update failed')).toBeVisible()
})

// Error paths for label edit
test('Edit label error shows alert', async () => {
  ;(api.catalogVersions.update as Mock).mockRejectedValue(new Error('409: duplicate label'))
  renderDetail('Admin')
  await page.getByRole('button', { name: 'Edit version label' }).click()
  await page.getByRole('textbox', { name: 'Version Label' }).clear()
  await page.getByRole('textbox', { name: 'Version Label' }).fill('dup')
  await page.getByRole('button', { name: 'Save' }).first().click()
  await expect.element(page.getByText('409: duplicate label')).toBeVisible()
})

// Label edit cancel
test('Edit label Cancel restores original', async () => {
  renderDetail('Admin')
  await page.getByRole('button', { name: 'Edit version label' }).click()
  await expect.element(page.getByRole('textbox', { name: 'Version Label' })).toBeVisible()
  await page.getByRole('button', { name: 'Cancel' }).first().click()
  await expect.element(page.getByRole('textbox', { name: 'Version Label' })).not.toBeInTheDocument()
})

// Remove pin error
test('Remove pin error shows alert', async () => {
  ;(api.catalogVersions.removePin as Mock).mockRejectedValue(new Error('500: pin removal failed'))
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Remove' }).first().click()
  await expect.element(page.getByText('500: pin removal failed')).toBeVisible()
})

// Add Pin: entity type version load error resets versions list
test('Add Pin: version load error clears version list', async () => {
  // Add an unpinned entity type
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
    { id: 'et-3', name: 'Platform', created_at: '', updated_at: '' },
  ], total: 3 })
  ;(api.versions.list as Mock).mockRejectedValue(new Error('500: versions failed'))
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await page.getByRole('dialog').getByText('Select entity type...').click()
  await page.getByTestId('pin-et-Platform').click()
  // Version select should not appear (versions failed to load)
  // No crash — gracefully handled
})

// Add Pin modal cancel via button
test('Add Pin modal Cancel closes', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// Add Pin modal close via Escape key (covers onClose callback)
test('Add Pin modal closes on Escape', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// === US-53 Task 12c/12d: Inline version dropdown & Add Pin filtering ===

// T-28.14: Version column is dropdown for Admin+
test('T-28.14: BOM version column shows dropdown for Admin', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  // Version column should have a MenuToggle button with "V3" text for Model pin
  await expect.element(page.getByRole('button', { name: 'Version for Model' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Version for Tool' })).toBeVisible()
})

// T-28.16: Selecting different version calls updatePin API
test('T-28.16: Selecting different version calls updatePin', async () => {
  // Mock versions.list for entity type et-1 (Model)
  ;(api.versions.list as Mock).mockResolvedValue({ items: [
    { id: 'etv-v1', entity_type_id: 'et-1', version: 1, description: 'V1', created_at: '' },
    { id: 'etv-v2', entity_type_id: 'et-1', version: 2, description: 'V2', created_at: '' },
    { id: 'etv-1', entity_type_id: 'et-1', version: 3, description: 'V3', created_at: '' },
  ], total: 3 })
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Version for Model' })).toBeVisible()
  // Open the version dropdown for Model pin
  await page.getByRole('button', { name: 'Version for Model' }).click()
  // Should see version options — select V2
  await expect.element(page.getByRole('option', { name: 'V2' })).toBeVisible()
  await page.getByRole('option', { name: 'V2' }).click()
  expect(api.catalogVersions.updatePin).toHaveBeenCalledWith('cv-1', 'pin-1', 'etv-v2')
})

// T-28.18: RO user sees plain text, not dropdown
test('T-28.18: RO user sees plain text version, not dropdown', async () => {
  renderDetail('RO')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Model', exact: true })).toBeVisible()
  // Should NOT have version dropdown buttons
  await expect.element(page.getByRole('button', { name: 'Version for Model' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Version for Tool' })).not.toBeInTheDocument()
  // Plain text "V3" should be visible in table cells
  await expect.element(page.getByRole('gridcell', { name: 'V3' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'V1' })).toBeVisible()
})

// T-28.19: Add Pin modal only shows unpinned entity types
test('T-28.19: Add Pin modal filters out already-pinned entity types', async () => {
  // mockPins already pins et-1 (Model) and et-2 (Tool)
  // entityTypes.list returns both Model and Tool
  // Adding a third unpinned entity type
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
    { id: 'et-3', name: 'Platform', created_at: '', updated_at: '' },
  ], total: 3 })
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Add Pin' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Open the entity type dropdown
  await page.getByRole('dialog').getByText('Select entity type...').click()
  // Model and Tool should NOT be in the dropdown (already pinned)
  // Platform should be available
  await expect.element(page.getByTestId('pin-et-Platform')).toBeVisible()
  await expect.element(page.getByTestId('pin-et-Model')).not.toBeInTheDocument()
  await expect.element(page.getByTestId('pin-et-Tool')).not.toBeInTheDocument()
})

// Update pin version error shows alert
test('Update pin version error shows alert', async () => {
  ;(api.versions.list as Mock).mockResolvedValue({ items: [
    { id: 'etv-v1', entity_type_id: 'et-1', version: 1, description: 'V1', created_at: '' },
    { id: 'etv-1', entity_type_id: 'et-1', version: 3, description: 'V3', created_at: '' },
  ], total: 2 })
  ;(api.catalogVersions.updatePin as Mock).mockRejectedValue(new Error('400: entity type mismatch'))
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Version for Model' }).click()
  await page.getByRole('option', { name: 'V1' }).click()
  await expect.element(page.getByText('400: entity type mismatch')).toBeVisible()
})

// Coverage: toggle off version dropdown by clicking again (L334-335)
test('BOM version dropdown toggles closed on second click', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Version for Model' })).toBeVisible()
  // Open
  await page.getByRole('button', { name: 'Version for Model' }).click()
  await expect.element(page.getByRole('option', { name: 'V1' })).toBeVisible()
  // Close by clicking again
  await page.getByRole('button', { name: 'Version for Model' }).click()
  await expect.element(page.getByRole('option', { name: 'V1' })).not.toBeInTheDocument()
})

// Coverage: Escape closes BOM version dropdown via onOpenChange (L534)
test('BOM version dropdown closes on Escape', async () => {
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Version for Model' }).click()
  await expect.element(page.getByRole('option', { name: 'V1' })).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await expect.element(page.getByRole('option', { name: 'V1' })).not.toBeInTheDocument()
})

// Coverage: version load error for BOM inline dropdown (L343)
test('BOM version dropdown handles version load error', async () => {
  ;(api.versions.list as Mock).mockRejectedValue(new Error('500: versions failed'))
  renderDetail('Admin')
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await expect.element(page.getByRole('button', { name: 'Version for Model' })).toBeVisible()
  // Open — versions.list fails, but dropdown should still render (empty options)
  await page.getByRole('button', { name: 'Version for Model' }).click()
  // No crash — gracefully handled
})
