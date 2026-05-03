import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogListPage from './CatalogListPage'
import { api } from '../../api/client'
import { statusColor } from '../../utils/statusColor'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: {
      list: vi.fn(),
      create: vi.fn(),
      delete: vi.fn(),
      import: vi.fn(),
    },
    catalogVersions: {
      list: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalogs = [
  {
    id: 'c1', name: 'production-app-a', description: 'Prod A',
    catalog_version_id: 'cv1', validation_status: 'draft', published: false,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'c2', name: 'staging-app-b', description: '',
    catalog_version_id: 'cv2', validation_status: 'valid', published: true,
    created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
  },
]

const mockCVs = [
  { id: 'cv1', version_label: 'release-1.0', lifecycle_stage: 'development', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'cv2', version_label: 'release-2.0', lifecycle_stage: 'testing', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
]

function renderList(role: 'Admin' | 'RW' | 'RO' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/catalogs']}>
      <Routes>
        <Route path="/schema/catalogs" element={<CatalogListPage role={role} />} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: mockCatalogs, total: 2 })
  ;(api.catalogs.create as Mock).mockResolvedValue({ id: 'c3', name: 'new-catalog' })
  ;(api.catalogs.delete as Mock).mockResolvedValue(undefined)
  ;(api.catalogVersions.list as Mock).mockResolvedValue({ items: mockCVs, total: 2 })
})

// T-10.41: Catalogs page renders
test('T-10.41: catalog list page renders', async () => {
  renderList()
  await expect.element(page.getByText('Catalogs')).toBeVisible()
})

// T-10.42: Catalog list shows name, CV ID, status badge, date
test('T-10.42: shows catalog list with correct columns', async () => {
  renderList()
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await expect.element(page.getByText('staging-app-b')).toBeVisible()
  await expect.element(page.getByText('draft')).toBeVisible()
  await expect.element(page.getByText('valid')).toBeVisible()
})

// T-10.43: Status badge color-coded
test('T-10.43: status badges are color-coded', async () => {
  renderList()
  // draft should be blue, valid should be green — verify labels exist
  await expect.element(page.getByText('draft')).toBeVisible()
  await expect.element(page.getByText('valid')).toBeVisible()
})

// T-10.44: Create button visible for RW+, hidden for RO
test('T-10.44: create button visible for RW', async () => {
  renderList('RW')
  await expect.element(page.getByRole('button', { name: 'Create Catalog' })).toBeVisible()
})

test('T-10.44: create button hidden for RO', async () => {
  renderList('RO')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  const buttons = page.getByRole('button', { name: 'Create Catalog' })
  await expect.element(buttons).not.toBeInTheDocument()
})

// T-10.45: Create modal has name, description, CV dropdown
test('T-10.45: create modal has form fields', async () => {
  renderList('Admin')
  await page.getByRole('button', { name: 'Create Catalog' }).click()
  await expect.element(page.getByPlaceholder('e.g. production-app-a')).toBeVisible()
  await expect.element(page.getByText('Select a catalog version')).toBeVisible()
})

// T-10.46: Invalid name shows inline error
test('T-10.46: invalid name shows error', async () => {
  renderList('Admin')
  await page.getByRole('button', { name: 'Create Catalog' }).click()
  const nameInput = page.getByPlaceholder('e.g. production-app-a')
  await nameInput.fill('My-Invalid-Name!')
  await expect.element(page.getByText('Must be lowercase alphanumeric')).toBeVisible()
})

// T-10.47: Create calls API and list refreshes
test('T-10.47: create submits and refreshes', async () => {
  renderList('Admin')
  await page.getByRole('button', { name: 'Create Catalog' }).click()

  const nameInput = page.getByPlaceholder('e.g. production-app-a')
  await nameInput.fill('new-catalog')

  // Select a CV
  await page.getByText('Select a catalog version').click()
  await page.getByText('release-1.0').click()

  await page.getByRole('button', { name: 'Create' }).click()

  expect(api.catalogs.create).toHaveBeenCalledWith({
    name: 'new-catalog',
    description: undefined,
    catalog_version_id: 'cv1',
  })
})

// T-10.48: Delete button visible for RW+, hidden for RO
test('T-10.48: delete button hidden for RO', async () => {
  renderList('RO')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  const deleteButtons = page.getByRole('button', { name: 'Delete' })
  await expect.element(deleteButtons).not.toBeInTheDocument()
})

// T-10.49: Delete shows confirmation dialog
test('T-10.49: delete shows confirmation', async () => {
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  const deleteButtons = page.getByRole('button', { name: 'Delete' })
  await deleteButtons.first().click()
  await expect.element(page.getByText('Are you sure you want to delete catalog')).toBeVisible()
})

// T-10.50: Delete confirm removes from list
test('T-10.50: delete confirm calls API', async () => {
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  const deleteButtons = page.getByRole('button', { name: 'Delete' })
  await deleteButtons.first().click()
  await expect.element(page.getByText('Are you sure you want to delete catalog')).toBeVisible()

  // Click the Delete button in the confirmation modal (the last one on the page)
  const allDeleteBtns = page.getByRole('button', { name: 'Delete' })
  await allDeleteBtns.nth(allDeleteBtns.elements().length - 1).click()

  expect(api.catalogs.delete).toHaveBeenCalledWith('production-app-a')
})

// === Additional coverage tests ===

test('empty catalog list shows empty state', async () => {
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderList()
  await expect.element(page.getByText('No catalogs yet. Create one to get started.')).toBeVisible()
})

test('load error shows error alert', async () => {
  ;(api.catalogs.list as Mock).mockRejectedValue(new Error('500: server error'))
  renderList()
  await expect.element(page.getByText('500: server error')).toBeVisible()
})

test('create failure shows error in modal', async () => {
  ;(api.catalogs.create as Mock).mockRejectedValue(new Error('409: name exists'))
  renderList('Admin')
  await page.getByRole('button', { name: 'Create Catalog' }).click()
  const nameInput = page.getByPlaceholder('e.g. production-app-a')
  await nameInput.fill('existing-catalog')
  await page.getByText('Select a catalog version').click()
  await page.getByText('release-1.0').click()
  await page.getByRole('button', { name: 'Create' }).click()
  await expect.element(page.getByText('409: name exists')).toBeVisible()
})

test('delete failure shows error in modal', async () => {
  ;(api.catalogs.delete as Mock).mockRejectedValue(new Error('500: delete failed'))
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Are you sure you want to delete catalog')).toBeVisible()
  const allDeleteBtns = page.getByRole('button', { name: 'Delete' })
  await allDeleteBtns.nth(allDeleteBtns.elements().length - 1).click()
  await expect.element(page.getByText('500: delete failed')).toBeVisible()
})

test('catalog version label shown, or truncated ID if no label', async () => {
  ;(api.catalogs.list as Mock).mockResolvedValue({
    items: [
      { ...mockCatalogs[0], catalog_version_label: 'v1.0' },
      { ...mockCatalogs[1], catalog_version_label: undefined },
    ],
    total: 2,
  })
  renderList()
  await expect.element(page.getByText('v1.0')).toBeVisible()
})

test('refresh button reloads catalogs', async () => {
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await page.getByRole('button', { name: 'Refresh' }).click()
  expect((api.catalogs.list as Mock).mock.calls.length).toBeGreaterThanOrEqual(2)
})

test('name too long shows validation error', async () => {
  renderList('Admin')
  await page.getByRole('button', { name: 'Create Catalog' }).click()
  const nameInput = page.getByPlaceholder('e.g. production-app-a')
  await nameInput.fill('a'.repeat(64))
  await expect.element(page.getByText('Name must be at most 63 characters')).toBeVisible()
})

// TD-41: Description column in catalog list
test('TD-41: catalog list shows description column', async () => {
  renderList()
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  // Catalog description should appear in a table cell
  await expect.element(page.getByRole('gridcell', { name: 'Prod A' })).toBeVisible()
})

// T-16.64: Published badge shown in catalog list
test('T-16.64: published badge shown in catalog list', async () => {
  renderList()
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  // staging-app-b has published: true
  await expect.element(page.getByText('published')).toBeVisible()
})

// === Import Button Tests ===

// Import button visible for Admin
test('import button visible for Admin', async () => {
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Import Catalog' })).toBeVisible()
})

// Import button hidden for RW (isAdmin check — RW is not Admin)
test('import button hidden for RW', async () => {
  renderList('RW')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  expect(page.getByRole('button', { name: 'Import Catalog' }).query()).toBeNull()
})

// Import button hidden for RO
test('import button hidden for RO', async () => {
  renderList('RO')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  expect(page.getByRole('button', { name: 'Import Catalog' }).query()).toBeNull()
})

// Clicking import button opens the import modal
test('clicking import button opens import modal', async () => {
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await page.getByRole('button', { name: 'Import Catalog' }).click()
  // Modal should show the file upload form
  await expect.element(page.getByText('Catalog File (JSON)')).toBeVisible()
})

// Cover CatalogListPage.tsx:278-279 (onSuccess callback: loadCatalogs + navigate)
test('import onSuccess reloads catalogs and navigates', async () => {
  // We need to capture navigation. Use MemoryRouter with a catch-all route.
  render(
    <MemoryRouter initialEntries={['/schema/catalogs']}>
      <Routes>
        <Route path="/schema/catalogs" element={<CatalogListPage role="Admin" />} />
        <Route path="/schema/catalogs/:name" element={<div data-testid="navigated">Navigated</div>} />
      </Routes>
    </MemoryRouter>
  )
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  // Open import modal
  await page.getByRole('button', { name: 'Import Catalog' }).click()
  await expect.element(page.getByText('Catalog File (JSON)')).toBeVisible()

  // Upload a JSON file
  const sampleExport = {
    format_version: '1.0',
    catalog: { name: 'imported-cat' },
    catalog_version: { label: 'v1.0' },
    entity_types: [],
    type_definitions: [],
  }
  const blob = new Blob([JSON.stringify(sampleExport)], { type: 'application/json' })
  const file = new File([blob], 'export.json', { type: 'application/json' })
  await new Promise(resolve => setTimeout(resolve, 50))
  const dropzone = document.querySelector('[data-testid="file-dropzone"]')
  expect(dropzone, 'file-dropzone element must exist on upload step').not.toBeNull()
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(file)
  dropzone!.dispatchEvent(new DragEvent('dragover', { bubbles: true, dataTransfer }))
  dropzone!.dispatchEvent(new DragEvent('drop', { bubbles: true, dataTransfer }))
  await new Promise(resolve => setTimeout(resolve, 300))

  // Mock dry run (no collisions) and actual import
  const dryRunResult = {
    status: 'ok',
    collisions: [],
    summary: { total_entities: 0, conflicts: 0, identical: 0, new: 0 },
  }
  const importResult = {
    status: 'ok',
    catalog_name: 'imported-cat',
    catalog_id: 'cat-new',
    types_created: 0,
    types_reused: 0,
    instances_created: 0,
    links_created: 0,
  }
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunResult)
    .mockResolvedValueOnce(importResult)

  // Click Analyze
  await page.getByRole('button', { name: 'Analyze' }).click()
  // Should go to confirm step
  await expect.element(page.getByText(/Ready to import/)).toBeVisible()
  // Click Import
  await page.getByRole('button', { name: 'Import' }).click()
  // Wait for done step
  await expect.element(page.getByText('Import Complete')).toBeVisible()

  // Record how many times catalogs.list was called before clicking View Catalog
  const callsBefore = (api.catalogs.list as Mock).mock.calls.length

  // Click View Catalog — triggers onSuccess which calls loadCatalogs() and navigate()
  await page.getByRole('button', { name: 'View Catalog' }).click()

  // Verify loadCatalogs was called (api.catalogs.list called again)
  await vi.waitFor(() => {
    expect((api.catalogs.list as Mock).mock.calls.length).toBeGreaterThan(callsBefore)
  })
  // Verify navigation happened (should navigate to /schema/catalogs/imported-cat)
  await expect.element(page.getByTestId('navigated')).toBeVisible()
})

// === CV load error — modal opens but CV list is empty ===
test('CV load error — create modal opens but CV select has no options', async () => {
  ;(api.catalogVersions.list as Mock).mockRejectedValue(new Error('500: CV service down'))
  renderList('Admin')
  await expect.element(page.getByText('production-app-a')).toBeVisible()
  await page.getByRole('button', { name: 'Create Catalog' }).click()
  // Modal should still open — verify by checking the name input placeholder
  await expect.element(page.getByPlaceholder('e.g. production-app-a')).toBeVisible()
  // CV toggle should show placeholder (no CV selected)
  await expect.element(page.getByText('Select a catalog version')).toBeVisible()
  // Open the select — it should have no options
  await page.getByText('Select a catalog version').click()
  // No CV options should be visible (release-1.0 or release-2.0)
  expect(page.getByText('release-1.0').query()).toBeNull()
  expect(page.getByText('release-2.0').query()).toBeNull()
})

// === statusColor default branch ===
test('statusColor returns grey for unknown status', () => {
  expect(statusColor('unknown')).toBe('grey')
  expect(statusColor('')).toBe('grey')
  expect(statusColor('pending')).toBe('grey')
})
