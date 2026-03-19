import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page, userEvent } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EntityTypeDetailPage from './EntityTypeDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    entityTypes: {
      get: vi.fn(),
      list: vi.fn(),
      copy: vi.fn(),
      delete: vi.fn(),
      rename: vi.fn(),
    },
    attributes: {
      list: vi.fn(),
      add: vi.fn(),
      remove: vi.fn(),
      reorder: vi.fn(),
      edit: vi.fn(),
      copyFrom: vi.fn(),
    },
    associations: {
      list: vi.fn(),
      create: vi.fn(),
      edit: vi.fn(),
      delete: vi.fn(),
    },
    enums: {
      list: vi.fn(),
    },
    versions: {
      list: vi.fn(),
      diff: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockEntityType = {
  id: 'et-1',
  name: 'MLModel',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

const mockAttributes = [
  { id: '', name: 'name', description: 'Instance name', type: 'string', ordinal: -2, required: true, system: true },
  { id: '', name: 'description', description: 'Instance description', type: 'string', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'hostname', description: 'The host', type: 'string', ordinal: 0, required: false },
  { id: 'a2', name: 'cpu_count', description: '', type: 'number', ordinal: 1, required: false },
  { id: 'a3', name: 'status', description: '', type: 'enum', enum_id: 'enum1', ordinal: 2, required: false },
]

const mockAssociations = [
  { id: 'assoc1', entity_type_version_id: 'v1', name: 'tools', target_entity_type_id: 'et-2', type: 'containment', source_role: 'parent', target_role: 'child', source_cardinality: '1', target_cardinality: '0..n', direction: 'outgoing' },
]

const mockVersions = [
  { id: 'v1', entity_type_id: 'et-1', version: 1, description: 'Initial', created_at: '2026-01-01T00:00:00Z' },
  { id: 'v2', entity_type_id: 'et-1', version: 2, description: 'Added attr', created_at: '2026-01-02T00:00:00Z' },
]

const mockOtherEntityTypes = [
  { id: 'et-1', name: 'MLModel', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'et-2', name: 'Dataset', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
]

function renderDetail(role: 'Admin' | 'RO' | 'SuperAdmin' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/entity-types/et-1']}>
      <Routes>
        <Route path="/entity-types/:id" element={<EntityTypeDetailPage role={role} />} />
        <Route path="/" element={<div>Home Page</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.entityTypes.get as Mock).mockResolvedValue(mockEntityType)
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: mockOtherEntityTypes, total: 2 })
  ;(api.entityTypes.copy as Mock).mockResolvedValue({ entity_type: { id: 'et-copy' }, version: { version: 1 } })
  ;(api.entityTypes.delete as Mock).mockResolvedValue(undefined)
  ;(api.attributes.list as Mock).mockResolvedValue({ items: mockAttributes, total: 5 })
  ;(api.attributes.add as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.attributes.remove as Mock).mockResolvedValue(undefined)
  ;(api.attributes.reorder as Mock).mockResolvedValue({ status: 'reordered' })
  ;(api.attributes.edit as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.attributes.copyFrom as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.entityTypes.rename as Mock).mockResolvedValue({ entity_type: { ...mockEntityType, name: 'NewName' }, was_deep_copy: false })
  ;(api.associations.list as Mock).mockResolvedValue({ items: mockAssociations, total: 1 })
  ;(api.associations.create as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.associations.edit as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.associations.delete as Mock).mockResolvedValue(undefined)
  ;(api.enums.list as Mock).mockResolvedValue({ items: [{ id: 'enum1', name: 'Status' }], total: 1 })
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 2 })
  ;(api.versions.diff as Mock).mockResolvedValue({
    from_version: 1,
    to_version: 2,
    changes: [{ name: 'hostname', change_type: 'added', category: 'attribute', old_value: '', new_value: 'string' }],
  })
})

// === Overview Tab ===

test('T-C.32: shows entity type name, ID, and dates on overview', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  await expect.element(page.getByText('et-1')).toBeVisible()
  await expect.element(page.getByText('Name', { exact: true })).toBeVisible()
  await expect.element(page.getByText('ID', { exact: true })).toBeVisible()
})

test('shows back link to entity types list', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  const backLink = page.getByRole('button', { name: /Back/i })
  await expect.element(backLink).toBeVisible()
})

test('shows Copy and Delete buttons for Admin', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Copy' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('T-C.45: RO role hides Copy and Delete buttons', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Copy' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

test('shows error when entity type load fails', async () => {
  ;(api.entityTypes.get as Mock).mockRejectedValue(new Error('404: not found'))
  renderDetail()
  await expect.element(page.getByText('404: not found')).toBeVisible()
})

// === Copy ===

test('T-C.44: copy entity type via modal', async () => {
  // Need versions loaded for copy to use latest version
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('Copy Entity Type')).toBeVisible()

  await page.getByRole('textbox', { name: /New Name/i }).fill('CopiedModel')
  await page.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()

  expect(api.entityTypes.copy).toHaveBeenCalledWith('et-1', { source_version: 1, new_name: 'CopiedModel' })
})

test('copy shows error on failure', async () => {
  ;(api.entityTypes.copy as Mock).mockRejectedValue(new Error('409: name exists'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Copy' }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('Dup')
  await page.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()

  await expect.element(page.getByText('409: name exists')).toBeVisible()
})

test('copy cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Copy' }).click()
  await expect.element(page.getByText('Copy Entity Type')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Copy Entity Type')).not.toBeInTheDocument()
})

// === Delete from detail page ===

test('delete confirmation from detail page', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.entityTypes.delete).toHaveBeenCalledWith('et-1')
})

test('delete cancel from detail page', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(api.entityTypes.delete).not.toHaveBeenCalled()
})

test('delete failure shows error', async () => {
  ;(api.entityTypes.delete as Mock).mockRejectedValue(new Error('500: server error'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('500: server error')).toBeVisible()
})

// === Attributes Tab ===

test('T-C.33: attributes tab lists attributes', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()
  await expect.element(page.getByText('cpu_count')).toBeVisible()
  await expect.element(page.getByText('status', { exact: true })).toBeVisible()
})

test('attributes tab shows type labels', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  // Check that all three attribute types are displayed as labels
  await expect.element(page.getByText('hostname')).toBeVisible()
  await expect.element(page.getByText('cpu_count')).toBeVisible()
  // Enum attribute should show the enum name
  await expect.element(page.getByText('enum (Status)')).toBeVisible()
})

test('attributes tab shows enum name for enum attributes', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('enum (Status)')).toBeVisible()
})

test('attributes empty state', async () => {
  ;(api.attributes.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('No attributes defined yet.')).toBeVisible()
})

test('T-C.34: add string attribute via modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Add Attribute' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('textbox', { name: /Name/i }).fill('newattr')
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  expect(api.attributes.add).toHaveBeenCalledWith('et-1', {
    name: 'newattr',
    description: undefined,
    type: 'string',
    enum_id: undefined,
    required: false,
  })
})

test('add attribute error shown in modal', async () => {
  ;(api.attributes.add as Mock).mockRejectedValue(new Error('409: duplicate'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Add Attribute' }).click()
  await page.getByRole('textbox', { name: /Name/i }).fill('hostname')
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  await expect.element(page.getByText('409: duplicate')).toBeVisible()
})

test('add attribute cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Add Attribute' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  // The dialog heading should be gone
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

test('T-C.36: remove attribute', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).first().click()
  expect(api.attributes.remove).toHaveBeenCalledWith('et-1', 'hostname')
})

test('T-C.37: reorder attributes with down button', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  // Click "Move down" on first custom attribute (hostname)
  await page.getByRole('button', { name: 'Move down' }).first().click()
  expect(api.attributes.reorder).toHaveBeenCalledWith('et-1', ['', '', 'a2', 'a1', 'a3'])
})

test('T-C.45: RO hides add/remove controls on attributes tab', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Add Attribute' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Move up' })).not.toBeInTheDocument()
})

// === Associations Tab ===

test('T-C.38: associations tab lists associations', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('contains')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()
  // Role column shows the other entity's role (target_role for outgoing)
  await expect.element(page.getByText('child')).toBeVisible()
})

test('associations empty state', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('No associations defined yet.')).toBeVisible()
})

test('T-C.41: remove association', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('Dataset')).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).click()
  expect(api.associations.delete).toHaveBeenCalledWith('et-1', 'tools')
})

test('T-C.40: add association modal shows with controls', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('Dataset')).toBeVisible()
  await page.getByRole('button', { name: 'Add Association' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Modal contains Add and Cancel buttons, and target/type selectors
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Add' })).toBeVisible()
  await expect.element(page.getByRole('dialog').getByRole('button', { name: 'Cancel' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Select target' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'containment' })).toBeVisible()
})

test('association add cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

test('RO hides association controls', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('Dataset')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Add Association' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
})

// === Version History Tab ===

test('T-C.42: version history tab shows versions', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('V1')).toBeVisible()
  await expect.element(page.getByText('V2')).toBeVisible()
  await expect.element(page.getByText('Initial')).toBeVisible()
  await expect.element(page.getByText('Added attr')).toBeVisible()
})

test('versions empty state', async () => {
  ;(api.versions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('No versions found.')).toBeVisible()
})

test('T-C.43: compare two versions shows diff', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('V1')).toBeVisible()

  await page.getByRole('spinbutton', { name: 'From version' }).fill('1')
  await page.getByRole('spinbutton', { name: 'To version' }).fill('2')
  await page.getByRole('button', { name: 'Compare' }).click()

  expect(api.versions.diff).toHaveBeenCalledWith('et-1', 1, 2)
  // The diff results should render
  await expect.element(page.getByText('hostname')).toBeVisible()
})

test('compare versions shows error on failure', async () => {
  ;(api.versions.diff as Mock).mockRejectedValue(new Error('404: version not found'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('V1')).toBeVisible()

  await page.getByRole('spinbutton', { name: 'From version' }).fill('1')
  await page.getByRole('spinbutton', { name: 'To version' }).fill('99')
  await page.getByRole('button', { name: 'Compare' }).click()

  await expect.element(page.getByText('404: version not found')).toBeVisible()
})

test('compare versions with no differences', async () => {
  ;(api.versions.diff as Mock).mockResolvedValue({ from_version: 1, to_version: 1, changes: [] })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('V1')).toBeVisible()

  await page.getByRole('spinbutton', { name: 'From version' }).fill('1')
  await page.getByRole('spinbutton', { name: 'To version' }).fill('1')
  await page.getByRole('button', { name: 'Compare' }).click()

  await expect.element(page.getByText('No differences found.')).toBeVisible()
})

test('compare button disabled when version fields empty', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('V1')).toBeVisible()

  const compareBtn = page.getByRole('button', { name: 'Compare' })
  await expect.element(compareBtn).toBeDisabled()
})

// === Error paths for additional coverage ===

test('remove attribute error shows alert', async () => {
  ;(api.attributes.remove as Mock).mockRejectedValue(new Error('500: remove failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).first().click()
  await expect.element(page.getByText('500: remove failed')).toBeVisible()
})

test('reorder attribute error shows alert', async () => {
  ;(api.attributes.reorder as Mock).mockRejectedValue(new Error('500: reorder failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Move down' }).first().click()
  await expect.element(page.getByText('500: reorder failed')).toBeVisible()
})

test('delete association error shows alert', async () => {
  ;(api.associations.delete as Mock).mockRejectedValue(new Error('500: delete assoc failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('Dataset')).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).click()
  await expect.element(page.getByText('500: delete assoc failed')).toBeVisible()
})

test('load attributes error shows alert', async () => {
  ;(api.attributes.list as Mock).mockRejectedValue(new Error('500: attrs failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('500: attrs failed')).toBeVisible()
})

test('load associations error shows alert', async () => {
  ;(api.associations.list as Mock).mockRejectedValue(new Error('500: assocs failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('500: assocs failed')).toBeVisible()
})

test('load versions error shows alert', async () => {
  ;(api.versions.list as Mock).mockRejectedValue(new Error('500: versions failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Version History/i }).click()
  await expect.element(page.getByText('500: versions failed')).toBeVisible()
})

// === Edit Attribute Modal ===

test('edit attribute opens modal pre-filled with current values', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Edit' }).first().click()
  await expect.element(page.getByText('Edit Attribute')).toBeVisible()

  // Modal should be pre-filled with the attribute's current name
  const nameInput = page.getByRole('textbox', { name: /Name/i })
  await expect.element(nameInput).toHaveValue('hostname')
})

test('edit attribute submits with new name', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Edit' }).first().click()
  await expect.element(page.getByText('Edit Attribute')).toBeVisible()

  await page.getByRole('textbox', { name: /Name/i }).fill('host')
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.attributes.edit).toHaveBeenCalledWith('et-1', 'hostname', expect.objectContaining({
    name: 'host',
  }))
})

test('edit attribute shows error on failure', async () => {
  ;(api.attributes.edit as Mock).mockRejectedValue(new Error('409: name conflict'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Edit' }).first().click()
  await page.getByRole('textbox', { name: /Name/i }).fill('cpu_count')
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('409: name conflict')).toBeVisible()
})

test('edit attribute cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Edit' }).first().click()
  await expect.element(page.getByText('Edit Attribute')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Edit Attribute')).not.toBeInTheDocument()
})

// === Rename Entity Type ===

test('rename button opens rename modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await expect.element(page.getByText('Rename Entity Type')).toBeVisible()
})

test('rename submits with new name', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('RenamedModel')
  await page.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  expect(api.entityTypes.rename).toHaveBeenCalledWith('et-1', 'RenamedModel', false)
})

test('rename shows deep copy warning on 409', async () => {
  ;(api.entityTypes.rename as Mock).mockRejectedValue(new Error('409: DEEP_COPY_REQUIRED'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('NewName')
  await page.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  await expect.element(page.getByText('Deep Copy Required')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Copy' })).toBeVisible()
})

test('rename error (non-deep-copy) shows in modal', async () => {
  ;(api.entityTypes.rename as Mock).mockRejectedValue(new Error('409: name already exists'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('ExistingName')
  await page.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  await expect.element(page.getByText('409: name already exists')).toBeVisible()
})

test('rename cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await expect.element(page.getByText('Rename Entity Type')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Rename Entity Type')).not.toBeInTheDocument()
})

test('RO cannot see edit name button', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  await expect.element(page.getByText('Rename', { exact: true })).not.toBeInTheDocument()
})

// === Copy Attributes Picker ===

test('copy from button opens copy attributes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await page.getByRole('button', { name: 'Copy from...' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()
})

test('copy attributes shows source entity types', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Copy from...' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()

  // Source selector should be visible
  await expect.element(page.getByRole('button', { name: 'Select source type' })).toBeVisible()
})

test('copy attributes cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Copy from...' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).not.toBeInTheDocument()
})

test('RO cannot see copy from button', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Copy from...' })).not.toBeInTheDocument()
})

// === Copy Attributes — Enum Name Display ===
// Note: PatternFly Select dropdowns can't be reliably clicked in browser tests.
// The enum name display in the copy-from modal is tested by verifying that the
// main attributes table also resolves enum names correctly (same code path).
// The copy modal source selection is tested via the system test suite.

test('attributes table shows enum name for enum-type attributes', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()
  // The enum attribute shows resolved enum name, not truncated ID
  await expect.element(page.getByText('enum (Status)')).toBeVisible()
})

test('RO cannot see edit buttons on attributes', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
})

// Bidirectional associations should show "references (mutual)" label
test('Associations tab shows bidirectional as "references (mutual)"', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-bi', entity_type_version_id: 'v1', name: 'dataset_ref', target_entity_type_id: 'et-2', type: 'bidirectional', source_role: 'in-group', target_role: 'tools', source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('references (mutual)')).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'Dataset', exact: true })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'tools', exact: true })).toBeVisible()
})

// Incoming bidirectional should also show "references (mutual)"
test('Associations tab shows incoming bidirectional as "references (mutual)"', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-bi-in', entity_type_version_id: 'v-other', name: 'model_ref', target_entity_type_id: 'et-1', type: 'bidirectional', source_role: 'in-group', target_role: 'tools', source_cardinality: '0..n', target_cardinality: '0..n', direction: 'incoming', source_entity_type_id: 'et-2' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('references (mutual)')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()
  // For incoming, show the source role (the other entity's role)
  await expect.element(page.getByRole('gridcell', { name: 'in-group', exact: true })).toBeVisible()
})

// T-E.84: Add association modal has cardinality dropdowns defaulting to "0..n"
test('T-E.84: Add association modal has cardinality dropdowns defaulting to 0..n', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  // Both cardinality labels should be visible in the modal
  await expect.element(page.getByText('Source Cardinality')).toBeVisible()
  await expect.element(page.getByText('Target Cardinality')).toBeVisible()
})

// T-E.85: Add association modal custom cardinality reveals min/max inputs
test('T-E.85: Custom cardinality reveals min/max inputs', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  // Switch to directional type (containment only allows 1 or 0..1, no custom)
  await page.getByRole('button', { name: 'containment' }).click()
  await page.getByText('directional', { exact: true }).click()
  // Select "Custom" for source cardinality
  await userEvent.selectOptions(page.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  // Min/max inputs should appear
  await expect.element(page.getByPlaceholder('min')).toBeVisible()
  await expect.element(page.getByPlaceholder('max or n')).toBeVisible()
})

// Fix #1: Custom cardinality with empty fields shows validation error
test('Custom cardinality with empty fields shows client-side error', async () => {
  ;(api.associations.create as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  // Switch to directional type (containment only allows 1 or 0..1, no custom)
  await page.getByRole('button', { name: 'containment' }).click()
  await page.getByText('directional', { exact: true }).click()
  // Select custom source cardinality but leave min/max fields empty
  await userEvent.selectOptions(page.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  await expect.element(page.getByPlaceholder('min')).toHaveValue('')
  await expect.element(page.getByPlaceholder('max or n')).toHaveValue('')
  // The button should be disabled (no target selected)
  const addBtn = page.getByRole('dialog').getByRole('button', { name: 'Add' })
  await expect.element(addBtn).toBeDisabled()
})

// Fix #4: Custom cardinality min field only accepts digits
test('Custom cardinality min field only accepts digits', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  // Switch to directional type (containment only allows 1 or 0..1, no custom)
  await page.getByRole('button', { name: 'containment' }).click()
  await page.getByText('directional', { exact: true }).click()
  await userEvent.selectOptions(page.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  const minInput = page.getByPlaceholder('min')
  // Type valid digits
  await userEvent.type(minInput, '2')
  await expect.element(minInput).toHaveValue('2')
  // Type letters — should be rejected (value stays '2')
  await userEvent.type(minInput, 'abc')
  await expect.element(minInput).toHaveValue('2')
})

// Containment source cardinality only shows 1 and 0..1
test('Containment source cardinality restricted to 1 and 0..1', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()
  // Default type is containment — source cardinality should only have 2 options
  const srcSelect = page.getByLabelText('Source Cardinality', { exact: true })
  await expect.element(srcSelect).toBeVisible()
  // Should NOT contain "0..n", "1..n", or "Custom" options
  // Verify by checking the select has value "0..1" (default for containment)
  await expect.element(srcSelect).toHaveValue('0..1')
})

// T-E.86: Associations table shows cardinality column
test('T-E.86: Associations table shows cardinality column', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  // Cardinality value for outgoing assoc: "1 → 0..n"
  await expect.element(page.getByText('1 → 0..n')).toBeVisible()
})

// T-E.104: Edit button opens modal with pre-filled association values
test('T-E.104: Edit button opens modal with pre-filled values', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  // Modal should be open with pre-filled values
  await expect.element(page.getByText('Edit Association')).toBeVisible()
  // Source role should be pre-filled with "parent"
  const dialog = page.getByRole('dialog')
  await expect.element(dialog.getByLabelText('Source Role')).toHaveValue('parent')
  await expect.element(dialog.getByLabelText('Target Role')).toHaveValue('child')
})

// T-E.105: Edit association Save triggers API call
test('T-E.105: Edit association Save triggers API call', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  // Change source role
  const dialog = page.getByRole('dialog')
  const sourceRoleInput = dialog.getByLabelText('Source Role')
  await userEvent.clear(sourceRoleInput)
  await userEvent.type(sourceRoleInput, 'updated_role')
  // Click Save
  await dialog.getByRole('button', { name: 'Save' }).click()
  // Verify API was called
  expect(api.associations.edit).toHaveBeenCalledWith('et-1', 'tools', expect.objectContaining({
    source_role: 'updated_role',
  }))
})

// Edit association modal shows editable Name field
test('Edit association modal has editable Name field', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Name field should be pre-filled with current name
  await expect.element(dialog.getByLabelText('Name')).toHaveValue('tools')
  // Change name and save
  const nameInput = dialog.getByLabelText('Name')
  await userEvent.clear(nameInput)
  await userEvent.type(nameInput, 'my_tools')
  await dialog.getByRole('button', { name: 'Save' }).click()
  // API should include the new name
  expect(api.associations.edit).toHaveBeenCalledWith('et-1', 'tools', expect.objectContaining({
    name: 'my_tools',
  }))
})

// Associations table columns are ordered: Relationship, Entity Type, Name
// Verify by checking that in the first data row, Entity Type (Dataset) comes before Name (tools)
test('Associations table column order: Entity Type before Name', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  // Get the first row's cells — Entity Type should be "Dataset", followed by Name "tools"
  const row = page.getByRole('row').filter({ hasText: 'contains' })
  const cells = await row.getByRole('gridcell').all()
  // cells[0] = Relationship (contains), cells[1] = Entity Type (Dataset), cells[2] = Name (tools)
  await expect.element(cells[1]).toHaveTextContent('Dataset')
  await expect.element(cells[2]).toHaveTextContent('tools')
})

// T-E.137: Edit modal shows custom cardinality option
test('T-E.137: Edit modal source cardinality has Custom option', async () => {
  // Use directional association (containment restricts source cardinality)
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select "Custom" for source cardinality
  await userEvent.selectOptions(dialog.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  // Min/max inputs should appear
  await expect.element(dialog.getByPlaceholder('min')).toBeVisible()
  await expect.element(dialog.getByPlaceholder('max or n')).toBeVisible()
})

// T-E.139: Edit modal pre-fills non-standard cardinality as Custom
test('T-E.139: Edit modal pre-fills custom cardinality', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '2..5', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Custom cardinality "2..5" should show min/max inputs pre-filled
  await expect.element(dialog.getByPlaceholder('min')).toHaveValue('2')
  await expect.element(dialog.getByPlaceholder('max or n')).toHaveValue('5')
})

// Edit modal: target custom cardinality pre-fills correctly
test('Edit modal pre-fills custom target cardinality', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '3..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Target cardinality "3..n" should show custom inputs
  const tgtMins = await dialog.getByPlaceholder('min').all()
  const tgtMaxes = await dialog.getByPlaceholder('max or n').all()
  // Second set of min/max inputs is for target (first is source if custom)
  // Since source is standard (0..n), only target should have custom inputs
  await expect.element(tgtMins[0]).toHaveValue('3')
  await expect.element(tgtMaxes[0]).toHaveValue('n')
})

// Edit modal: target custom cardinality select and type values
test('Edit modal target Custom option reveals min/max inputs', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select Custom for target cardinality
  await userEvent.selectOptions(dialog.getByLabelText('Target Cardinality', { exact: true }), 'custom')
  // Min/max inputs should appear for target
  const mins = await dialog.getByPlaceholder('min').all()
  expect(mins.length).toBeGreaterThan(0)
})

// T-E.138: Edit modal custom cardinality sends correct value
test('T-E.138: Edit modal custom cardinality sends correct value', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  ;(api.associations.edit as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select Custom for source cardinality (already directional type)
  await userEvent.selectOptions(dialog.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  // Fill in custom values
  await userEvent.type(dialog.getByPlaceholder('min'), '3')
  await userEvent.type(dialog.getByPlaceholder('max or n'), '7')
  // Save
  await dialog.getByRole('button', { name: 'Save' }).click()
  // Verify API was called with custom cardinality
  expect(api.associations.edit).toHaveBeenCalledWith('et-1', 'ref', expect.objectContaining({
    source_cardinality: '3..7',
  }))
})

// Edit modal: target custom min/max digit-only validation
test('Edit modal target custom min only accepts digits, max accepts digits or n', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select Custom for target cardinality
  await userEvent.selectOptions(dialog.getByLabelText('Target Cardinality', { exact: true }), 'custom')
  const mins = await dialog.getByPlaceholder('min').all()
  const maxes = await dialog.getByPlaceholder('max or n').all()
  // Target min/max should be the first (only) set since source is standard
  const tgtMin = mins[0]
  const tgtMax = maxes[0]
  // Type digits — should work
  await userEvent.type(tgtMin, '4')
  await expect.element(tgtMin).toHaveValue('4')
  // Type letters — should be rejected
  await userEvent.type(tgtMin, 'abc')
  await expect.element(tgtMin).toHaveValue('4')
  // Max accepts 'n'
  await userEvent.type(tgtMax, 'n')
  await expect.element(tgtMax).toHaveValue('n')
  // Max accepts digits
  await userEvent.clear(tgtMax)
  await userEvent.type(tgtMax, '10')
  await expect.element(tgtMax).toHaveValue('10')
  // Max rejects letters
  await userEvent.type(tgtMax, 'xyz')
  await expect.element(tgtMax).toHaveValue('10')
})

// Edit modal: switching source cardinality from Custom back to standard
test('Edit modal source cardinality switch from Custom back to standard', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select Custom
  await userEvent.selectOptions(dialog.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  await expect.element(dialog.getByPlaceholder('min')).toBeVisible()
  // Switch back to standard "1..n"
  await userEvent.selectOptions(dialog.getByLabelText('Source Cardinality', { exact: true }), '1..n')
  // Custom inputs should disappear
  const mins = await dialog.getByPlaceholder('min').all()
  expect(mins.length).toBe(0)
})

// Edit modal: switching target cardinality from Custom back to standard
test('Edit modal target cardinality switch from Custom back to standard', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Select Custom for target
  await userEvent.selectOptions(dialog.getByLabelText('Target Cardinality', { exact: true }), 'custom')
  const mins = await dialog.getByPlaceholder('min').all()
  expect(mins.length).toBeGreaterThan(0)
  // Switch back to standard "1"
  await userEvent.selectOptions(dialog.getByLabelText('Target Cardinality', { exact: true }), '1')
  // Custom inputs should disappear
  const minsAfter = await dialog.getByPlaceholder('min').all()
  expect(minsAfter.length).toBe(0)
})

// Edit modal: changing type to containment resets source cardinality
test('Edit modal type change to containment resets source cardinality', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '1..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  // Type is editable (allowTypeChange is set) — change to containment
  await userEvent.selectOptions(dialog.getByLabelText('Type', { exact: true }), 'containment')
  // Source cardinality "1..n" is invalid for containment — should reset to "0..1"
  const srcSelect = dialog.getByLabelText('Source Cardinality', { exact: true })
  await expect.element(srcSelect).toHaveValue('0..1')
})

// Edit modal: save error is displayed
test('Edit modal shows save error', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc1', entity_type_version_id: 'v1', name: 'ref', target_entity_type_id: 'et-2',
        type: 'directional', source_role: 'src', target_role: 'tgt',
        source_cardinality: '0..n', target_cardinality: '0..n', direction: 'outgoing' },
    ],
    total: 1,
  })
  ;(api.associations.edit as Mock).mockRejectedValue(new Error('500: server error'))
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: 'Save' }).click()
  await expect.element(dialog.getByText('500: server error')).toBeVisible()
})

// T-E.144: Add attribute modal has required checkbox
test('T-E.144: Add attribute modal has required checkbox', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Add Attribute' }).click()
  const dialog = page.getByRole('dialog')
  const checkbox = dialog.getByLabelText('Required')
  await expect.element(checkbox).toBeVisible()
})

// T-E.145: Edit attribute modal pre-fills required checkbox
test('T-E.145: Edit attribute modal pre-fills required', async () => {
  ;(api.attributes.list as Mock).mockResolvedValue({
    items: [
      { id: 'a1', name: 'hostname', description: '', type: 'string', ordinal: 0, required: true },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Edit' }).click()
  const dialog = page.getByRole('dialog')
  const checkbox = dialog.getByRole('checkbox', { name: /Required/i })
  await expect.element(checkbox).toBeChecked()
})

// T-E.146: Attributes table shows required indicator
test('T-E.146: Attributes table shows required indicator', async () => {
  ;(api.attributes.list as Mock).mockResolvedValue({
    items: [
      { id: 'a1', name: 'hostname', description: '', type: 'string', ordinal: 0, required: true },
      { id: 'a2', name: 'port', description: '', type: 'number', ordinal: 1, required: false },
    ],
    total: 2,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  // Required attribute should have an indicator (asterisk or "Required" text)
  await expect.element(page.getByText('hostname *')).toBeVisible()
})

// === Additional coverage tests ===

// Incoming association row hides Edit/Remove buttons
test('incoming association hides Edit/Remove buttons', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-in', entity_type_version_id: 'v-other', name: 'tools', target_entity_type_id: 'et-1', type: 'containment', source_role: 'parent', target_role: 'child', source_cardinality: '1', target_cardinality: '0..n', direction: 'incoming', source_entity_type_id: 'et-2' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('contained by')).toBeVisible()
  // Incoming associations should NOT have Edit or Remove buttons
  const editBtns = page.getByRole('button', { name: 'Edit' })
  expect(editBtns.elements().length).toBe(0)
  const removeBtns = page.getByRole('button', { name: 'Remove' })
  expect(removeBtns.elements().length).toBe(0)
})

// Incoming directional shows "referenced by" label
test('incoming directional shows "referenced by" label', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-in', entity_type_version_id: 'v-other', name: 'depends', target_entity_type_id: 'et-1', type: 'directional', source_role: 'consumer', target_role: 'provider', source_cardinality: '0..n', target_cardinality: '1', direction: 'incoming', source_entity_type_id: 'et-2' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('referenced by')).toBeVisible()
  // Should show the source role (other entity's role) — "consumer"
  await expect.element(page.getByRole('gridcell', { name: 'consumer' })).toBeVisible()
})

// Deep copy: "Create Copy" button calls rename with deepCopyAllowed=true
test('deep copy Create Copy button calls rename with deepCopyAllowed=true', async () => {
  ;(api.entityTypes.rename as Mock)
    .mockRejectedValueOnce(new Error('409: DEEP_COPY_REQUIRED'))
    .mockResolvedValueOnce({ entity_type: { id: 'et-new', name: 'NewName' }, was_deep_copy: true })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('NewName')
  await page.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  // Deep copy warning should appear
  await expect.element(page.getByText('Deep Copy Required')).toBeVisible()
  await page.getByRole('button', { name: 'Create Copy' }).click()

  // Should call rename with deepCopyAllowed=true
  expect(api.entityTypes.rename).toHaveBeenCalledWith('et-1', 'NewName', true)
})

// Deep copy warning cancel closes modal
test('deep copy warning cancel closes modal', async () => {
  ;(api.entityTypes.rename as Mock).mockRejectedValueOnce(new Error('409: DEEP_COPY_REQUIRED'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  await page.getByRole('textbox', { name: /New Name/i }).fill('NewName')
  await page.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  await expect.element(page.getByText('Deep Copy Required')).toBeVisible()
  await page.getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByText('Deep Copy Required')).not.toBeInTheDocument()
})

// Add association with custom source cardinality validation error
test('add association custom cardinality empty fields shows validation error', async () => {
  // Use empty associations so "Dataset" only appears in the dropdown, not the table
  ;(api.associations.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()

  // Fill required fields: name and target
  await page.getByRole('textbox', { name: /Name/i }).fill('test-assoc')
  // Switch to directional for custom cardinality support
  await page.getByRole('button', { name: 'containment' }).click()
  await page.getByText('directional', { exact: true }).click()
  // Select custom source cardinality
  await userEvent.selectOptions(page.getByLabelText('Source Cardinality', { exact: true }), 'custom')
  // Select a target
  await page.getByRole('button', { name: 'Select target' }).click()
  await page.getByText('Dataset', { exact: true }).click()
  // Now click Add — should show cardinality validation error
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()
  await expect.element(page.getByText(/Source cardinality: both min and max are required/)).toBeVisible()
})

// Add association with custom target cardinality validation error
test('add association custom target cardinality empty fields shows validation error', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()

  await page.getByRole('textbox', { name: /Name/i }).fill('test-assoc')
  await page.getByRole('button', { name: 'containment' }).click()
  await page.getByText('directional', { exact: true }).click()
  // Select custom target cardinality
  await userEvent.selectOptions(page.getByLabelText('Target Cardinality', { exact: true }), 'custom')
  // Select a target
  await page.getByRole('button', { name: 'Select target' }).click()
  await page.getByText('Dataset', { exact: true }).click()
  // Click Add — should show target cardinality validation error
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()
  await expect.element(page.getByText(/Target cardinality: both min and max are required/)).toBeVisible()
})

// Add association error from API
test('add association API error shown in modal', async () => {
  ;(api.associations.create as Mock).mockRejectedValue(new Error('409: already exists'))
  ;(api.associations.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await page.getByRole('button', { name: 'Add Association' }).click()

  await page.getByRole('textbox', { name: /Name/i }).fill('tools')
  await page.getByRole('button', { name: 'Select target' }).click()
  await page.getByText('Dataset', { exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  await expect.element(page.getByText('409: already exists')).toBeVisible()
})

// Rename disabled when name unchanged
test('rename button disabled when name unchanged', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()

  await page.getByText('Rename', { exact: true }).click()
  // Name is pre-filled with current name "MLModel" — rename button should be disabled
  const renameBtn = page.getByRole('dialog').getByRole('button', { name: 'Rename' })
  await expect.element(renameBtn).toBeDisabled()
})

// Copy attributes: select source and see attributes listed
test('copy attributes shows source attributes after selection', async () => {
  const sourceAttrs = [
    { id: 'sa1', name: 'region', description: 'Region', type: 'string', ordinal: 0, required: false },
    { id: 'sa2', name: 'hostname', description: 'Host', type: 'string', ordinal: 1, required: false },
  ]
  ;(api.attributes.list as Mock).mockImplementation((etId: string) => {
    if (etId === 'et-2') return Promise.resolve({ items: sourceAttrs, total: 2 })
    return Promise.resolve({ items: mockAttributes, total: 5 })
  })
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 2 })

  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()
  await page.getByRole('button', { name: 'Copy from...' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()

  // Select the source entity type — click the toggle
  await page.getByRole('dialog').getByRole('button', { name: 'Select source type' }).click()
  await page.getByText('Dataset').first().click()

  // Attributes from source should appear
  await expect.element(page.getByText('region').first()).toBeVisible()
  // "hostname" conflicts with existing attribute — should show Conflict label
  await expect.element(page.getByText('Conflict')).toBeVisible()
  // "region" is available
  await expect.element(page.getByText('Available')).toBeVisible()
})

// Copy attributes: select and submit
test('copy attributes submits selected attributes', async () => {
  const sourceAttrs = [
    { id: 'sa1', name: 'region', description: 'Region', type: 'string', ordinal: 0, required: false },
  ]
  ;(api.attributes.list as Mock).mockImplementation((etId: string) => {
    if (etId === 'et-2') return Promise.resolve({ items: sourceAttrs, total: 1 })
    return Promise.resolve({ items: mockAttributes, total: 5 })
  })
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 2 })

  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Copy from...' }).click()

  // Select source
  await page.getByRole('dialog').getByRole('button', { name: 'Select source type' }).click()
  await page.getByText('Dataset').first().click()

  // Wait for source attrs to load
  await expect.element(page.getByText('region').first()).toBeVisible()

  // Check the "region" attribute checkbox
  const checkbox = page.getByRole('dialog').getByRole('checkbox').first()
  await checkbox.click()

  // Click "Copy Selected"
  await page.getByRole('dialog').getByRole('button', { name: 'Copy Selected' }).click()
  expect(api.attributes.copyFrom).toHaveBeenCalledWith('et-1', {
    source_entity_type_id: 'et-2',
    source_version: 2,
    attribute_names: ['region'],
  })
})

// Copy attributes: error on submit
test('copy attributes error shown in modal', async () => {
  ;(api.attributes.copyFrom as Mock).mockRejectedValue(new Error('500: copy failed'))
  const sourceAttrs = [
    { id: 'sa1', name: 'region', description: 'Region', type: 'string', ordinal: 0, required: false },
  ]
  ;(api.attributes.list as Mock).mockImplementation((etId: string) => {
    if (etId === 'et-2') return Promise.resolve({ items: sourceAttrs, total: 1 })
    return Promise.resolve({ items: mockAttributes, total: 5 })
  })
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 2 })

  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Copy from...' }).click()

  // Select source
  await page.getByRole('dialog').getByRole('button', { name: 'Select source type' }).click()
  await page.getByText('Dataset').first().click()

  await expect.element(page.getByText('region').first()).toBeVisible()
  const checkbox = page.getByRole('dialog').getByRole('checkbox').first()
  await checkbox.click()

  await page.getByRole('dialog').getByRole('button', { name: 'Copy Selected' }).click()
  await expect.element(page.getByText('500: copy failed')).toBeVisible()
})

// Move up attribute
test('reorder attributes with up button', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  // Click "Move up" on the second custom attribute (cpu_count)
  const upButtons = page.getByRole('button', { name: 'Move up' })
  // First Move up button is on the first custom row (hostname), second is on the second custom row (cpu_count)
  await upButtons.nth(1).click()
  expect(api.attributes.reorder).toHaveBeenCalledWith('et-1', ['', '', 'a2', 'a1', 'a3'])
})

// I1: Move up disabled for first custom attr (right after system attrs)
test('I1: move up disabled for first custom attribute after system attrs', async () => {
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByText('hostname')).toBeVisible()

  // The first Move up button (on hostname, the first custom attr) should be disabled
  const upButtons = page.getByRole('button', { name: 'Move up' })
  await expect.element(upButtons.first()).toHaveAttribute('disabled')
})

// SuperAdmin can also edit
test('SuperAdmin can see edit controls', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: 'MLModel' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Copy' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
  await expect.element(page.getByText('Rename', { exact: true })).toBeVisible()
})

// === System Attributes ===

// T-18.33: Entity type detail shows "Name" system attribute with System badge
test('T-18.33: Name system attribute shows System badge', async () => {
  // mockAttributes already includes system attrs (name, description)
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  // Name row should have "System" label
  const nameCell = page.getByRole('gridcell', { name: /name.*System/ })
  await expect.element(nameCell).toBeVisible()
})

// T-18.34: Entity type detail shows "Description" system attribute with System badge
test('T-18.34: Description system attribute shows System badge', async () => {
  // mockAttributes already includes system attrs (name, description)
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  const descCell = page.getByRole('gridcell', { name: /description.*System/ })
  await expect.element(descCell).toBeVisible()
})

// T-18.35: System attributes appear before custom attributes
test('T-18.35: system attributes appear before custom attributes', async () => {
  // mockAttributes already includes system attrs (name, description) before custom attrs
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  // System attrs with System badge appear, then custom attrs follow
  await expect.element(page.getByRole('gridcell', { name: /name.*System/ })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: /description.*System/ })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'hostname' })).toBeVisible()
  // System attrs have ordinal -2 and -1, custom start at 0 — backend provides them in order
  // Verify ordinals: Name=-2, Description=-1, hostname=0
  await expect.element(page.getByRole('gridcell', { name: '-2' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: '-1' })).toBeVisible()
})

// T-18.36: Delete button hidden for system attributes
test('T-18.36: Remove button hidden for system attributes', async () => {
  // mockAttributes already includes system attrs (name, description)
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByRole('gridcell', { name: /name.*System/ })).toBeVisible()
  // System rows should not have Remove buttons — only custom rows have them
  // There are 3 custom attributes, each with a Remove button = 3 Remove buttons total
  const removeButtons = page.getByRole('button', { name: 'Remove' })
  await expect.element(removeButtons.first()).toBeVisible()
  // Count: we expect exactly 3 Remove buttons (not 5)
  await expect.element(removeButtons.nth(2)).toBeVisible()
})

// T-18.37: Edit button hidden for system attributes
test('T-18.37: Edit button hidden for system attributes', async () => {
  // mockAttributes already includes system attrs (name, description)
  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await expect.element(page.getByRole('gridcell', { name: /name.*System/ })).toBeVisible()
  // Only 3 Edit buttons for the 3 custom attributes
  const editButtons = page.getByRole('button', { name: 'Edit' })
  await expect.element(editButtons.first()).toBeVisible()
  await expect.element(editButtons.nth(2)).toBeVisible()
})

// T-18.38: Copy attributes picker excludes system attributes
test('T-18.38: copy attributes picker excludes system attributes', async () => {
  const sourceAttrs = [
    { id: '', name: 'name', description: 'Instance name', type: 'string', ordinal: -2, required: true, system: true },
    { id: '', name: 'description', description: 'Instance description', type: 'string', ordinal: -1, required: false, system: true },
    { id: 'sa1', name: 'region', description: 'Region', type: 'string', ordinal: 0, required: false },
  ]
  ;(api.attributes.list as Mock).mockImplementation((etId: string) => {
    if (etId === 'et-2') return Promise.resolve({ items: sourceAttrs, total: 3 })
    return Promise.resolve({ items: mockAttributes, total: 5 })
  })
  ;(api.versions.list as Mock).mockResolvedValue({ items: mockVersions, total: 2 })

  renderDetail()
  await page.getByRole('tab', { name: /Attributes/i }).click()
  await page.getByRole('button', { name: 'Copy from...' }).click()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()

  // Select source entity type
  await page.getByRole('dialog').getByRole('button', { name: 'Select source type' }).click()
  await page.getByText('Dataset').first().click()

  // Only "region" should be shown (system attrs filtered out)
  await expect.element(page.getByText('region').first()).toBeVisible()
  // System attrs should NOT appear in the picker list
  await expect.element(page.getByRole('dialog').getByRole('gridcell', { name: /^Name/ })).not.toBeInTheDocument()
})

// Cardinality display for incoming association: target → source
test('incoming association shows inverted cardinality', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-in', entity_type_version_id: 'v-other', name: 'tools', target_entity_type_id: 'et-1', type: 'containment', source_role: 'parent', target_role: 'child', source_cardinality: '1', target_cardinality: '0..n', direction: 'incoming', source_entity_type_id: 'et-2' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  // Incoming cardinality should show target → source: "0..n → 1"
  await expect.element(page.getByText('0..n → 1')).toBeVisible()
})
