import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter } from 'react-router-dom'
import App from './App'
import { api, setAuthRole } from './api/client'

vi.mock('./api/client', () => ({
  api: {
    entityTypes: {
      list: vi.fn(),
      create: vi.fn(),
      delete: vi.fn(),
      containmentTree: vi.fn(),
    },
    catalogVersions: {
      list: vi.fn(),
      create: vi.fn(),
      promote: vi.fn(),
      demote: vi.fn(),
      delete: vi.fn(),
    },
    enums: {
      list: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockEntityTypes = [
  { id: 'et-1', name: 'MLModel', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'et-2', name: 'Dataset', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
]

const mockCatalogVersions = [
  { id: 'cv-1', version_label: 'v1.0', lifecycle_stage: 'development', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cv-2', version_label: 'v2.0', lifecycle_stage: 'testing', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
  { id: 'cv-3', version_label: 'v3.0', lifecycle_stage: 'production', created_at: '2026-01-03T00:00:00Z', updated_at: '2026-01-03T00:00:00Z' },
]

// Containment tree: Server (root) → Tool (child) → Subcomponent (grandchild), plus standalone Dataset
const mockContainmentTree = [
  {
    entity_type: { id: 'et-server', name: 'Server', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
    versions: [
      { id: 'vs1', entity_type_id: 'et-server', version: 1, description: 'V1', created_at: '2026-01-01T00:00:00Z' },
    ],
    latest_version: 1,
    children: [
      {
        entity_type: { id: 'et-tool', name: 'Tool', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
        versions: [
          { id: 'vt1', entity_type_id: 'et-tool', version: 1, description: 'V1', created_at: '2026-01-01T00:00:00Z' },
          { id: 'vt2', entity_type_id: 'et-tool', version: 2, description: 'V2', created_at: '2026-01-02T00:00:00Z' },
        ],
        latest_version: 2,
        children: [
          {
            entity_type: { id: 'et-sub', name: 'Subcomponent', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
            versions: [
              { id: 'vsc1', entity_type_id: 'et-sub', version: 1, description: 'V1', created_at: '2026-01-01T00:00:00Z' },
            ],
            latest_version: 1,
            children: [],
          },
        ],
      },
    ],
  },
  {
    entity_type: { id: 'et-dataset', name: 'Dataset', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
    versions: [
      { id: 'vd1', entity_type_id: 'et-dataset', version: 1, description: 'V1', created_at: '2026-01-01T00:00:00Z' },
      { id: 'vd2', entity_type_id: 'et-dataset', version: 2, description: 'V2', created_at: '2026-01-02T00:00:00Z' },
      { id: 'vd3', entity_type_id: 'et-dataset', version: 3, description: 'V3', created_at: '2026-01-03T00:00:00Z' },
    ],
    latest_version: 3,
    children: [],
  },
]

function renderApp(initialPath = '/') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <App />
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: mockEntityTypes, total: 2 })
  ;(api.entityTypes.create as Mock).mockResolvedValue({ entity_type: { id: 'et-3', name: 'NewType' } })
  ;(api.entityTypes.delete as Mock).mockResolvedValue(undefined)
  ;(api.catalogVersions.list as Mock).mockResolvedValue({ items: mockCatalogVersions, total: 3 })
  ;(api.catalogVersions.create as Mock).mockResolvedValue({ id: 'cv-4', version_label: 'v4.0', lifecycle_stage: 'development' })
  ;(api.catalogVersions.promote as Mock).mockResolvedValue({ status: 'promoted' })
  ;(api.catalogVersions.demote as Mock).mockResolvedValue({ status: 'demoted' })
  ;(api.catalogVersions.delete as Mock).mockResolvedValue(undefined)
  if (api.enums?.list) {
    ;(api.enums.list as Mock).mockResolvedValue({ items: [], total: 0 })
  }
  if (api.entityTypes?.containmentTree) {
    ;(api.entityTypes.containmentTree as Mock).mockResolvedValue(mockContainmentTree)
  }
})

// === Entity Types: Rendering ===

test('renders heading and fetches entity types on mount', async () => {
  renderApp()
  await expect.element(page.getByText('AI Asset Hub')).toBeVisible()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()
  expect(setAuthRole).toHaveBeenCalledWith('Admin')
})

test('shows empty state when no entity types', async () => {
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderApp()
  await expect.element(page.getByText('No entity types yet. Create one to get started.')).toBeVisible()
})

test('shows error alert when list API fails', async () => {
  ;(api.entityTypes.list as Mock).mockRejectedValue(new Error('network error'))
  renderApp()
  await expect.element(page.getByText('network error')).toBeVisible()
})

test('shows Create and Delete buttons for Admin role', async () => {
  renderApp()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' }).first()).toBeVisible()
})

// === Entity Types: Filtering ===

test('filters entity types by name', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()

  await page.getByPlaceholder('Filter by name').fill('ML')
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByText('Dataset')).not.toBeInTheDocument()
})

test('shows no-match message when filter has no results', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByPlaceholder('Filter by name').fill('zzzzz')
  await expect.element(page.getByText('No entity types match the filter.')).toBeVisible()
})

// === Entity Types: Create ===

test('opens create modal and creates entity type', async () => {
  renderApp()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()

  await page.getByRole('button', { name: 'Create Entity Type' }).click()

  // Modal is open with form fields
  await expect.element(page.getByRole('textbox', { name: /Name/i })).toBeVisible()
  await page.getByRole('textbox', { name: /Name/i }).fill('NewType')
  await page.getByRole('textbox', { name: /Description/i }).fill('A new type')

  // Click Create in the modal
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.entityTypes.create).toHaveBeenCalledWith({
    name: 'NewType',
    description: 'A new type',
  })
})

test('shows error in create modal when API fails', async () => {
  ;(api.entityTypes.create as Mock).mockRejectedValue(new Error('409: name already exists'))
  renderApp()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()

  await page.getByRole('button', { name: 'Create Entity Type' }).click()
  await page.getByRole('textbox', { name: /Name/i }).fill('Duplicate')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await expect.element(page.getByText('409: name already exists')).toBeVisible()
})

test('closes create modal on Cancel', async () => {
  renderApp()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()

  await page.getByRole('button', { name: 'Create Entity Type' }).click()
  await expect.element(page.getByRole('textbox', { name: /Name/i })).toBeVisible()

  await page.getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('textbox', { name: /Name/i })).not.toBeInTheDocument()
})

// === Entity Types: Delete with Confirmation (T-C.29 through T-C.31) ===

test('T-C.29: click Delete shows confirmation modal with entity type name', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  // Verify modal body mentions the entity type name
  await expect.element(page.getByRole('dialog').getByText('MLModel')).toBeVisible()
})

test('T-C.30: cancel confirmation does nothing', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  // Cancel
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

  // Entity type still present, delete not called
  await expect.element(page.getByText('MLModel')).toBeVisible()
  expect(api.entityTypes.delete).not.toHaveBeenCalled()
})

test('T-C.31: confirm deletion calls API and removes entity', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  // Confirm
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.entityTypes.delete).toHaveBeenCalledWith('et-1')
})

test('shows error when delete fails', async () => {
  ;(api.entityTypes.delete as Mock).mockRejectedValue(new Error('500: internal error'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('500: internal error')).toBeVisible()
})

// === Entity Types: Refresh ===

test('refreshes entity types on Refresh click', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  const callsBefore = (api.entityTypes.list as Mock).mock.calls.length
  await page.getByRole('button', { name: 'Refresh' }).click()

  // wait for the new call
  await vi.waitFor(() => {
    expect((api.entityTypes.list as Mock).mock.calls.length).toBeGreaterThan(callsBefore)
  })
})

// === Entity Types: Navigation ===

test('entity type name is clickable', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  // Name should be a link button
  const nameLink = page.getByRole('button', { name: 'MLModel' })
  await expect.element(nameLink).toBeVisible()
})

// === Catalog Versions: Tab Switching ===

test('switches to Catalog Versions tab and displays data', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('tab', { name: /Catalog Versions/i }).click()

  await expect.element(page.getByText('v1.0')).toBeVisible()
  await expect.element(page.getByText('v2.0')).toBeVisible()
  await expect.element(page.getByText('v3.0')).toBeVisible()
  await expect.element(page.getByText('development')).toBeVisible()
  await expect.element(page.getByText('testing')).toBeVisible()
  await expect.element(page.getByText('production')).toBeVisible()
})

test('shows correct Promote/Demote buttons per lifecycle stage', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  // Admin: 2 Promote buttons (dev→test, test→prod), 1 Demote button (testing only, not production)
  const promoteButtons = page.getByRole('button', { name: 'Promote' })
  const demoteButtons = page.getByRole('button', { name: 'Demote' })
  await expect.element(promoteButtons.nth(0)).toBeVisible()
  await expect.element(promoteButtons.nth(1)).toBeVisible()
  await expect.element(demoteButtons.nth(0)).toBeVisible()
  // Production Demote should NOT be visible for Admin
  await expect.element(demoteButtons.nth(1)).not.toBeInTheDocument()
})

test('Admin cannot see Demote on production catalog version', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v3.0')).toBeVisible()

  // The production CV row should NOT have a Demote button for Admin
  const prodRow = page.getByRole('row').filter({ hasText: 'v3.0' })
  await expect.element(prodRow.getByRole('button', { name: 'Demote' })).not.toBeInTheDocument()

  // Testing CV row SHOULD have Demote for Admin
  const testRow = page.getByRole('row').filter({ hasText: 'v2.0' })
  await expect.element(testRow.getByRole('button', { name: 'Demote' })).toBeVisible()
})

test('shows empty state for catalog versions', async () => {
  ;(api.catalogVersions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('No catalog versions yet. Create one to get started.')).toBeVisible()
})

// === Catalog Versions: Promote/Demote ===

test('promotes a catalog version', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Promote' }).first().click()
  expect(api.catalogVersions.promote).toHaveBeenCalledWith('cv-1')
})

test('demotes a testing version to development', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v2.0')).toBeVisible()

  await page.getByRole('button', { name: 'Demote' }).first().click()
  expect(api.catalogVersions.demote).toHaveBeenCalledWith('cv-2', 'development')
})

test('demotes a production version to testing (SuperAdmin)', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  // Switch to SuperAdmin — only SuperAdmin can demote from production
  await page.getByRole('button', { name: /Role: Admin/i }).click()
  await page.getByRole('option', { name: 'SuperAdmin' }).click()
  await expect.element(page.getByRole('button', { name: /Role: SuperAdmin/i })).toBeVisible()

  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v3.0')).toBeVisible()

  // SuperAdmin sees Demote on production row
  const prodRow = page.getByRole('row').filter({ hasText: 'v3.0' })
  await prodRow.getByRole('button', { name: 'Demote' }).click()
  expect(api.catalogVersions.demote).toHaveBeenCalledWith('cv-3', 'testing')
})

test('shows error when promote fails', async () => {
  ;(api.catalogVersions.promote as Mock).mockRejectedValue(new Error('403: forbidden'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Promote' }).first().click()
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

test('shows error when demote fails', async () => {
  ;(api.catalogVersions.demote as Mock).mockRejectedValue(new Error('403: forbidden'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v2.0')).toBeVisible()

  await page.getByRole('button', { name: 'Demote' }).first().click()
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

// === Catalog Versions: Create ===

test('creates catalog version via modal', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await page.getByPlaceholder('e.g. v1.0').fill('v4.0')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.catalogVersions.create).toHaveBeenCalledWith({ version_label: 'v4.0' })
})

test('shows error when catalog version create fails', async () => {
  ;(api.catalogVersions.create as Mock).mockRejectedValue(new Error('409: duplicate label'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await page.getByPlaceholder('e.g. v1.0').fill('v1.0')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await expect.element(page.getByText('409: duplicate label')).toBeVisible()
})

// === Role Switching ===

test('switching to RO hides Create and Delete buttons', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()

  // Open role dropdown and select RO
  await page.getByRole('button', { name: /Role: Admin/i }).click()
  await page.getByRole('option', { name: 'RO' }).click()

  await expect.element(page.getByRole('button', { name: /Role: RO/i })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  expect(setAuthRole).toHaveBeenCalledWith('RO')
})

test('switching to SuperAdmin keeps Create and Delete visible', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: /Role: Admin/i }).click()
  await page.getByRole('option', { name: 'SuperAdmin' }).click()

  await expect.element(page.getByRole('button', { name: /Role: SuperAdmin/i })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()
  expect(setAuthRole).toHaveBeenCalledWith('SuperAdmin')
})

// === Coverage: catalog version list error ===

test('shows error when catalog version list fails', async () => {
  ;(api.catalogVersions.list as Mock).mockRejectedValue(new Error('cv network error'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('cv network error')).toBeVisible()
})

// === Coverage: close modals via X button ===

test('closes entity type modal via X button', async () => {
  renderApp()
  await expect.element(page.getByRole('button', { name: 'Create Entity Type' })).toBeVisible()
  await page.getByRole('button', { name: 'Create Entity Type' }).click()
  await expect.element(page.getByRole('textbox', { name: /Name/i })).toBeVisible()

  // Close via the X button in the modal header
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByRole('textbox', { name: /Name/i })).not.toBeInTheDocument()
})

test('closes catalog version modal via Cancel', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByPlaceholder('e.g. v1.0')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByPlaceholder('e.g. v1.0')).not.toBeInTheDocument()
})

test('closes catalog version modal via X button', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByPlaceholder('e.g. v1.0')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByPlaceholder('e.g. v1.0')).not.toBeInTheDocument()
})

// === Coverage: SearchInput clear ===

test('clears search filter', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()

  await page.getByPlaceholder('Filter by name').fill('ML')
  await expect.element(page.getByText('Dataset')).not.toBeInTheDocument()

  // Clear the search via the clear button (reset icon inside SearchInput)
  await page.getByRole('button', { name: /Reset/i }).click()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()
})

test('RO role hides Promote/Demote on Catalog Versions tab', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  // Switch to RO
  await page.getByRole('button', { name: /Role: Admin/i }).click()
  await page.getByRole('option', { name: 'RO' }).click()
  await expect.element(page.getByRole('button', { name: /Role: RO/i })).toBeVisible()

  // Switch to Catalog Versions tab
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  // Promote/Demote/Create should not be visible
  await expect.element(page.getByRole('button', { name: 'Promote' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Demote' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Create Catalog Version' })).not.toBeInTheDocument()
})

// === Enums Tab ===

test('shows Enums tab', async () => {
  renderApp()
  await expect.element(page.getByRole('tab', { name: /Enums/i })).toBeVisible()
})

// === Catalog Version Delete ===

test('Admin sees Delete button on non-production catalog versions', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  // Should have Delete buttons for dev and testing versions
  const deleteButtons = page.getByRole('button', { name: 'Delete' })
  await expect.element(deleteButtons.first()).toBeVisible()
})

test('delete catalog version shows confirmation modal', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  // Click Delete on first catalog version (v1.0, development)
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('v1.0')).toBeVisible()
})

test('cancel catalog version deletion does nothing', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(api.catalogVersions.delete).not.toHaveBeenCalled()
})

test('confirm catalog version deletion calls API', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.catalogVersions.delete).toHaveBeenCalledWith('cv-1')
})

test('RO role hides catalog version Delete buttons', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()

  await page.getByRole('button', { name: /Role: Admin/i }).click()
  await page.getByRole('option', { name: 'RO' }).click()

  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

test('catalog version delete error shows alert', async () => {
  ;(api.catalogVersions.delete as Mock).mockRejectedValue(new Error('403: forbidden'))
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

// === CV Create Containment Tree Tests (T-E.55 through T-E.58) ===

test('T-E.55: CV create modal shows containment tree with indentation', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByPlaceholder('e.g. v1.0')).toBeVisible()

  // Tree nodes should be visible
  await expect.element(page.getByRole('dialog').getByText('Server')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Tool')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Dataset')).toBeVisible()
  // Subcomponent is a grandchild — visible in tree
  await expect.element(page.getByRole('dialog').getByText('Subcomponent')).toBeVisible()
})

test('T-E.56: Selecting parent auto-selects all descendants recursively', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByRole('dialog').getByText('Server')).toBeVisible()

  // Check Server → Tool and Subcomponent should auto-check
  await page.getByRole('dialog').getByRole('checkbox', { name: 'Server' }).click()

  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Server' })).toBeChecked()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Tool' })).toBeChecked()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Subcomponent' })).toBeChecked()
  // Dataset should not be affected
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Dataset' })).not.toBeChecked()
})

test('T-E.57: Deselecting parent deselects all descendants recursively', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByRole('dialog').getByText('Server')).toBeVisible()

  // Select Server first (selects all descendants)
  await page.getByRole('dialog').getByRole('checkbox', { name: 'Server' }).click()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Tool' })).toBeChecked()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Subcomponent' })).toBeChecked()

  // Deselect Server → all descendants deselected
  await page.getByRole('dialog').getByRole('checkbox', { name: 'Server' }).click()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Server' })).not.toBeChecked()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Tool' })).not.toBeChecked()
  await expect.element(page.getByRole('dialog').getByRole('checkbox', { name: 'Subcomponent' })).not.toBeChecked()
})

test('T-E.58: Version dropdown shows all versions, defaults to latest', async () => {
  renderApp()
  await expect.element(page.getByText('MLModel')).toBeVisible()
  await page.getByRole('tab', { name: /Catalog Versions/i }).click()
  await expect.element(page.getByText('v1.0')).toBeVisible()

  await page.getByRole('button', { name: 'Create Catalog Version' }).click()
  await expect.element(page.getByRole('dialog').getByText('Server')).toBeVisible()

  // Dataset has 3 versions — the version dropdown should exist with latest (V3) selected
  const datasetSelect = page.getByRole('dialog').getByRole('combobox', { name: 'Version for Dataset' })
  await expect.element(datasetSelect).toBeVisible()
  // The selected value should be vd3 (latest version ID for Dataset)
  await expect.element(datasetSelect).toHaveValue('vd3')

  // Tool has 2 versions — latest is V2
  const toolSelect = page.getByRole('dialog').getByRole('combobox', { name: 'Version for Tool' })
  await expect.element(toolSelect).toBeVisible()
  await expect.element(toolSelect).toHaveValue('vt2')
})
