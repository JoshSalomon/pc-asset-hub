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
    instances: { list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn() },
    enums: { listValues: vi.fn() },
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
  associations: [],
}

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
