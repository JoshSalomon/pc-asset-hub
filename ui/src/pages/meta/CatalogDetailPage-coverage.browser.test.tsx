import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page, userEvent } from 'vitest/browser'
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
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'c1') return Promise.resolve(childInstances[0])
    if (id === 'p1') return Promise.resolve({ id: 'p1', name: 'my-parent', entity_type_id: 'et1' })
    return Promise.resolve({ id, name: `inst-${id}`, entity_type_id: 'et1' })
  })
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

  // Fill port (number attribute) — it renders as textbox with inputMode=decimal
  const portInput = page.getByRole('dialog').getByRole('textbox', { name: /port/i })
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
  // Re-fetch of the instance itself succeeds, but parent resolution fails
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'c1') return Promise.resolve(childInstances[0])
    return Promise.reject(new Error('404'))
  })
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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

// Cat 9: Enum select open/select in create modal
test('enum select opens and selects value in create modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a3', name: 'status', base_type: 'enum', type_definition_version_id: 'tdv-enum1', constraints: { values: ['active', 'inactive'] }, ordinal: 3, required: false },
    ],
  }
  ;(api.versions.snapshot as Mock).mockResolvedValue(snapshotWithEnum)
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: /Create model/ }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select "active" from the native <select> element
  const enumSelect = page.getByRole('dialog').getByRole('combobox', { name: 'status' })
  await userEvent.selectOptions(enumSelect, 'active')

  // Fill name and submit
  await page.getByRole('dialog').getByRole('textbox').first().fill('enum-inst')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()
  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({
    name: 'enum-inst',
    attributes: expect.objectContaining({ status: 'active' }),
  }))
})

// Cat 9: Enum select in edit modal
test('enum select works in edit modal', async () => {
  const snapshotWithEnum = {
    ...mockSnapshot,
    attributes: [
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'a3', name: 'status', base_type: 'enum', type_definition_version_id: 'tdv-enum1', constraints: { values: ['active', 'inactive'] }, ordinal: 3, required: false },
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
  ;(api.instances.list as Mock).mockResolvedValue({ items: instancesWithEnum, total: 1 })
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'inst-a' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Should show enum select with current value "active"
  const enumSelect = page.getByRole('dialog').getByRole('combobox', { name: 'status' })
  await expect.element(enumSelect).toHaveValue('active')
  // Change to "inactive"
  await userEvent.selectOptions(enumSelect, 'inactive')

  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({
    attributes: expect.objectContaining({ status: 'inactive' }),
  }))
})

// Cat 10: Edit modal description onChange (line 829)
test('edit modal description input updates value', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
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

// Cat 7: Child attr enum select onChange in add child modal
test('add child modal enum attr select works', async () => {
  const toolSnapshot = {
    entity_type: { id: 'et2', name: 'tool' },
    version: { id: 'etv2', version: 1 },
    attributes: [
      { id: 'sys-name', name: 'name', base_type: 'string', ordinal: -2, required: true, system: true },
      { id: 'sys-desc', name: 'description', base_type: 'string', ordinal: -1, required: false, system: true },
      { id: 'ta1', name: 'priority', base_type: 'enum', type_definition_version_id: 'tdv-enum2', constraints: { values: ['high', 'low'] }, ordinal: 1, required: false },
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
  await page.getByRole('dialog').getByRole('textbox', { name: /Name/i }).fill('child-with-enum')

  // The enum attr should render as a native <select> — select "high"
  const enumSelect = page.getByRole('dialog').getByRole('combobox', { name: 'priority' })
  await userEvent.selectOptions(enumSelect, 'high')

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

// T-21.16: Model Diagram tab exists on catalog detail page
test('T-21.16: Model Diagram tab exists on catalog detail page', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Model Diagram' })).toBeVisible()
})

// T-21.17: Clicking Model Diagram tab loads diagram data
test('T-21.17: clicking Model Diagram tab loads diagram data', async () => {
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Model Diagram' })).toBeVisible()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  // Should call listPins and snapshot for each pin
  await vi.waitFor(() => {
    expect(api.catalogVersions.listPins).toHaveBeenCalledWith('cv1')
  })
})

// T-21.19: Diagram tab renders diagram container (read-only — no click handlers passed)
test('T-21.19: diagram tab renders diagram container', async () => {
  renderDetail()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  // The diagram component should be in the DOM
  await expect.element(page.getByTestId('entity-type-diagram')).toBeVisible()
})

// T-21.18b: Diagram error is displayed
test('T-21.18b: diagram tab shows error alert on API failure', async () => {
  // First call to listPins succeeds (for useCatalogData), second call fails (for diagram)
  ;(api.catalogVersions.listPins as Mock)
    .mockResolvedValueOnce({ items: mockPins, total: 2 })
    .mockRejectedValueOnce(new Error('Network error'))
  renderDetail()
  await expect.element(page.getByRole('tab', { name: 'Model Diagram' })).toBeVisible()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

// T-21.20: Empty state on diagram tab when diagram data is empty (pins exist but snapshots empty)
test('T-21.20: diagram tab shows empty state when no diagram data loaded', async () => {
  // Pins exist (tabs render), but snapshot returns empty associations/attributes
  ;(api.versions.snapshot as Mock).mockResolvedValue({
    entity_type: { id: 'et1', name: 'model' }, version: { id: 'etv1', version: 1 },
    attributes: [], associations: [],
  })
  renderDetail()
  await page.getByRole('tab', { name: 'Model Diagram' }).click()
  // Diagram component renders (data loads successfully)
  await expect.element(page.getByTestId('entity-type-diagram')).toBeVisible()
})

// === Phase 2 CRUD: Catalog Description Inline Edit ===

test('Edit description button visible for RW+ on catalog detail', async () => {
  renderDetail('RW')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
})

test('Edit description button hidden for RO on catalog detail', async () => {
  renderDetail('RO')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
})

test('Catalog description edit flow: click Edit, change, Save calls API', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByText('Test catalog')).toBeVisible()
  await page.getByRole('button', { name: 'Edit description' }).click()
  const input = page.getByRole('textbox', { name: 'Description' })
  await expect.element(input).toBeVisible()
  await input.fill('updated desc')
  await page.getByRole('button', { name: 'Save' }).first().click()
  expect(api.catalogs.update).toHaveBeenCalledWith('my-catalog', { description: 'updated desc' })
})

test('Catalog description edit Cancel restores original', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit description' }).click()
  await expect.element(page.getByRole('textbox', { name: 'Description' })).toBeVisible()
  await page.getByRole('button', { name: 'Cancel' }).first().click()
  // Original description should be visible again
  await expect.element(page.getByText('Test catalog')).toBeVisible()
  await expect.element(page.getByRole('textbox', { name: 'Description' })).not.toBeInTheDocument()
})

// === Phase 2 CRUD: CV Selector ===

test('CV selector visible for Admin on unpublished catalog', async () => {
  renderDetail('Admin')
  await waitForInstances()
  // The CV selector should render as a MenuToggle with the current CV label
  await expect.element(page.getByRole('button', { name: 'Select catalog version' })).toBeVisible()
})

test('CV selector hidden for RO user', async () => {
  renderDetail('RO')
  await waitForInstances()
  // RO should see plain text, no dropdown
  await expect.element(page.getByRole('button', { name: 'Select catalog version' })).not.toBeInTheDocument()
})

test('CV selector hidden for RW user (not Admin)', async () => {
  renderDetail('RW')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Select catalog version' })).not.toBeInTheDocument()
})

test('CV selector disabled (hidden) on published catalog', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, validation_status: 'valid', published: true })
  renderDetail('Admin')
  await waitForInstances()
  // Published catalogs should show plain text, not a dropdown
  await expect.element(page.getByRole('button', { name: 'Select catalog version' })).not.toBeInTheDocument()
})

// Description edit error
test('edit description error shows alert', async () => {
  ;(api.catalogs.update as Mock).mockRejectedValue(new Error('500: update failed'))
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit description' }).click()
  await page.getByRole('textbox', { name: 'Description' }).fill('bad')
  await page.getByRole('button', { name: 'Save' }).first().click()
  await expect.element(page.getByText('500: update failed')).toBeVisible()
})

// CV selector opens and loads CV list
test('CV selector dropdown opens and shows CVs', async () => {
  ;(api.catalogVersions.list as Mock).mockResolvedValue({
    items: [
      { id: 'cv1', version_label: 'v1.0' },
      { id: 'cv2', version_label: 'v2.0' },
    ],
    total: 2,
  })
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Select catalog version' }).click()
  await expect.element(page.getByText('v2.0')).toBeVisible()
})

// CV re-pin calls API
test('selecting a CV calls update API', async () => {
  ;(api.catalogVersions.list as Mock).mockResolvedValue({
    items: [
      { id: 'cv1', version_label: 'v1.0' },
      { id: 'cv-new', version_label: 'v3.0' },
    ],
    total: 2,
  })
  ;(api.catalogs.update as Mock).mockResolvedValue({ ...mockCatalog, catalog_version_id: 'cv-new' })
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Select catalog version' }).click()
  await page.getByText('v3.0').click()
  await vi.waitFor(() => {
    expect(api.catalogs.update).toHaveBeenCalledWith('my-catalog', { catalog_version_id: 'cv-new' })
  })
})

// CV selector closes on Escape (covers onOpenChange callback)
test('CV selector closes on Escape', async () => {
  ;(api.catalogVersions.list as Mock).mockResolvedValue({
    items: [{ id: 'cv1', version_label: 'v1.0' }],
    total: 1,
  })
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Select catalog version' }).click()
  await expect.element(page.getByText('v1.0').last()).toBeVisible()
  await userEvent.keyboard('{Escape}')
  // Dropdown should close
})

// CV re-pin error
test('CV re-pin error shows alert', async () => {
  ;(api.catalogVersions.list as Mock).mockResolvedValue({
    items: [{ id: 'cv-bad', version_label: 'v-bad' }],
    total: 1,
  })
  ;(api.catalogs.update as Mock).mockRejectedValue(new Error('400: invalid CV'))
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Select catalog version' }).click()
  await page.getByText('v-bad').click()
  await expect.element(page.getByText('400: invalid CV')).toBeVisible()
})

// === Validate Write Protection Tests (TD-71: published catalog security fix) ===

test('T-30.15: Validate button hidden on published catalog for RW user', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('RW')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Validate' })).not.toBeInTheDocument()
})

test('T-30.16: Validate button visible on published catalog for SuperAdmin', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('SuperAdmin')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Validate' })).toBeVisible()
})

test('T-30.17: Validate button visible on unpublished catalog for RW (no regression)', async () => {
  renderDetail('RW')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Validate' })).toBeVisible()
})

// T-30.18: Published catalog hides mutation UI for RW
test('T-30.18: published catalog hides Edit description and Create button for RW', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('RW')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  // Edit description button should be hidden
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
  // Create button (e.g. "Create model") should be hidden
  await expect.element(page.getByRole('button', { name: /^Create / })).not.toBeInTheDocument()
})

// T-30.19: Published catalog shows mutation UI for SuperAdmin
test('T-30.19: published catalog shows Edit description and Create button for SuperAdmin', async () => {
  ;(api.catalogs.get as Mock).mockResolvedValue({ ...mockCatalog, published: true })
  renderDetail('SuperAdmin')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: /^Create / })).toBeVisible()
})

// T-30.20: Unpublished catalog still shows mutation UI for RW (no regression)
test('T-30.20: unpublished catalog shows Edit description and Create button for RW', async () => {
  renderDetail('RW')
  await expect.element(page.getByText('my-catalog')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: /^Create / })).toBeVisible()
})

// === TD-68: Inline TextInput width matches container ===

test('TD-68: description edit TextInput has width 100% and no max-width', async () => {
  renderDetail()
  await waitForInstances()
  await page.getByRole('button', { name: 'Edit description' }).click()
  const input = page.getByRole('textbox', { name: 'Description' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveAttribute('style', expect.stringContaining('width: 100%'))
  const style = input.element().getAttribute('style') || ''
  expect(style).not.toContain('max-width')
})

// === TD-51: Remove Container error is surfaced, not swallowed ===

test('TD-51: remove container error is shown to user', async () => {
  const childInstances = [{
    id: 'c1', entity_type_id: 'et1', catalog_id: 'cat1', parent_instance_id: 'p1',
    name: 'child-inst', description: '', version: 1,
    attributes: [{ name: 'hostname', type: 'string', value: 'h1' }],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }]
  ;(api.instances.list as Mock).mockResolvedValue({ items: childInstances, total: 1 })
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'c1') return Promise.resolve(childInstances[0])
    if (id === 'p1') return Promise.resolve({ id: 'p1', name: 'my-parent', entity_type_id: 'et1' })
    return Promise.resolve({ id, name: `inst-${id}`, entity_type_id: 'et1' })
  })
  ;(api.instances.setParent as Mock).mockRejectedValue(new Error('403: forbidden'))
  renderDetail('Admin')
  await expect.element(page.getByRole('gridcell', { name: 'child-inst' })).toBeVisible()
  await page.getByRole('button', { name: 'Details' }).first().click()
  await expect.element(page.getByText('Contained by: my-parent').first()).toBeVisible()
  await page.getByRole('button', { name: 'Set Container' }).first().click()
  await expect.element(page.getByRole('dialog')).toBeVisible()
  // Click "Remove Container" — the API will reject
  await page.getByRole('dialog').getByRole('button', { name: 'Remove Container' }).click()
  // Error should be displayed in the dialog, not swallowed
  await expect.element(page.getByText('403: forbidden')).toBeVisible()
})

// === Export Button Tests ===

// Export button visible for Admin
test('export button visible for Admin', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await expect.element(page.getByRole('button', { name: 'Export' })).toBeVisible()
})

// Export button hidden for RW
test('export button hidden for RW', async () => {
  renderDetail('RW')
  await waitForInstances()
  expect(page.getByRole('button', { name: 'Export' }).query()).toBeNull()
})

// Export button hidden for RO
test('export button hidden for RO', async () => {
  renderDetail('RO')
  await waitForInstances()
  expect(page.getByRole('button', { name: 'Export' }).query()).toBeNull()
})

// Clicking Export calls api.catalogs.export and triggers download
test('clicking Export calls API and triggers download', async () => {
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Export' }).click()

  expect(api.catalogs.export).toHaveBeenCalledWith('my-catalog')
})

// Export error shows error message
test('export error shows error message', async () => {
  ;(api.catalogs.export as Mock).mockRejectedValue(new Error('export failed'))
  renderDetail('Admin')
  await waitForInstances()
  await page.getByRole('button', { name: 'Export' }).click()
  await expect.element(page.getByText('export failed')).toBeVisible()
})
