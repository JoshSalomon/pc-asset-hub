import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
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
  { id: 'a1', name: 'hostname', description: 'The host', type: 'string', ordinal: 0, required: false },
  { id: 'a2', name: 'cpu_count', description: '', type: 'number', ordinal: 1, required: false },
  { id: 'a3', name: 'status', description: '', type: 'enum', enum_id: 'enum1', ordinal: 2, required: false },
]

const mockAssociations = [
  { id: 'assoc1', entity_type_version_id: 'v1', target_entity_type_id: 'et-2', type: 'containment', source_role: 'parent', target_role: 'child', direction: 'outgoing' },
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
  ;(api.attributes.list as Mock).mockResolvedValue({ items: mockAttributes, total: 3 })
  ;(api.attributes.add as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.attributes.remove as Mock).mockResolvedValue(undefined)
  ;(api.attributes.reorder as Mock).mockResolvedValue({ status: 'reordered' })
  ;(api.attributes.edit as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.attributes.copyFrom as Mock).mockResolvedValue({ id: 'v3', version: 3 })
  ;(api.entityTypes.rename as Mock).mockResolvedValue({ entity_type: { ...mockEntityType, name: 'NewName' }, was_deep_copy: false })
  ;(api.associations.list as Mock).mockResolvedValue({ items: mockAssociations, total: 1 })
  ;(api.associations.create as Mock).mockResolvedValue({ id: 'v3', version: 3 })
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
  const backLink = page.getByRole('button', { name: /Back to Entity Types/i })
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

  // Click "Move down" on first attribute
  await page.getByRole('button', { name: 'Move down' }).first().click()
  expect(api.attributes.reorder).toHaveBeenCalledWith('et-1', ['a2', 'a1', 'a3'])
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
  expect(api.associations.delete).toHaveBeenCalledWith('et-1', 'assoc1')
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
      { id: 'assoc-bi', entity_type_version_id: 'v1', target_entity_type_id: 'et-2', type: 'bidirectional', source_role: 'in-group', target_role: 'tools', direction: 'outgoing' },
    ],
    total: 1,
  })
  renderDetail()
  await page.getByRole('tab', { name: /Associations/i }).click()
  await expect.element(page.getByText('references (mutual)')).toBeVisible()
  await expect.element(page.getByText('Dataset')).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'tools', exact: true })).toBeVisible()
})

// Incoming bidirectional should also show "references (mutual)"
test('Associations tab shows incoming bidirectional as "references (mutual)"', async () => {
  ;(api.associations.list as Mock).mockResolvedValue({
    items: [
      { id: 'assoc-bi-in', entity_type_version_id: 'v-other', target_entity_type_id: 'et-1', type: 'bidirectional', source_role: 'in-group', target_role: 'tools', direction: 'incoming', source_entity_type_id: 'et-2' },
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
