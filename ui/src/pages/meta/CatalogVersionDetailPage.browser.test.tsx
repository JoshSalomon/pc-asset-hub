import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
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
    },
    versions: {
      snapshot: vi.fn(),
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
  { entity_type_name: 'Model', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 3 },
  { entity_type_name: 'Tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
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
  await expect.element(page.getByRole('button', { name: 'Model' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Tool' })).toBeVisible()
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
      { entity_type_name: 'Model', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 3, description: 'ML model type' },
      { entity_type_name: 'Tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
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
  await expect.element(page.getByRole('button', { name: 'Model' })).toBeVisible()
  await page.getByRole('button', { name: 'Model' }).click()
  // Modal should open, NOT navigate away
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Model — V3')).toBeVisible()
})

// T-E.65: BOM modal shows attributes table
test('T-E.65: BOM modal shows attributes table', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model' }).click()
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
  await page.getByRole('button', { name: 'Model' }).click()
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
  await page.getByRole('button', { name: 'Model' }).click()
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
  await expect.element(page.getByRole('button', { name: 'Model' })).toBeVisible()
  await page.getByRole('button', { name: 'Model' }).click()
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
  await page.getByRole('button', { name: 'Model' }).click()
  // Verify cardinality values: outgoing containment "1 → 0..n"
  await expect.element(page.getByText('1 → 0..n')).toBeVisible()
})

// BOM modal associations show name column
test('BOM modal associations show name column', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Bill of Materials' }).click()
  await page.getByRole('button', { name: 'Model' }).click()
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
