import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import CatalogDetailPage from './CatalogDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn(), list: vi.fn(), validate: vi.fn(), publish: vi.fn(), unpublish: vi.fn(), copy: vi.fn(), replace: vi.fn(), update: vi.fn(), export: vi.fn() },
    catalogVersions: { listPins: vi.fn(), list: vi.fn() },
    versions: { snapshot: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), createContained: vi.fn(), listContained: vi.fn(), setParent: vi.fn() },
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

// === Open in Data Viewer link ===

test('catalog detail shows Open in Data Viewer link', async () => {
  renderDetail()
  await waitForInstances()
  const link = page.getByText('Open in Data Viewer')
  await expect.element(link).toBeVisible()
  // Clicking should navigate (no full page reload)
  await link.click()
  // Navigation happens — no crash
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
  await expect.element(page.getByText('published', { exact: true })).toBeVisible()
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

// Warning banner shown for Admin on published catalog (Admin can't edit published either)
test('warning banner shown for Admin on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByText('Editing requires SuperAdmin')).toBeVisible()
})

// No warning banner for SuperAdmin on published catalog (SuperAdmin CAN edit)
test('no warning banner for SuperAdmin on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('SuperAdmin')
  await waitForInstances()
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Name and Description should be present
  await expect.element(page.getByRole('dialog').getByText('Name *')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Description', { exact: true })).toBeVisible()
})

// T-18.44: Edit submits updated name/description as top-level request fields
test('T-18.44: edit submits name/description as top-level fields', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Name * *')).not.toBeInTheDocument()
})
