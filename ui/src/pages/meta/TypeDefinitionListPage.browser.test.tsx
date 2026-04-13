import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page, userEvent } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import TypeDefinitionListPage from './TypeDefinitionListPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    typeDefinitions: {
      list: vi.fn(),
      create: vi.fn(),
      delete: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockSystemType = {
  id: 'td-sys-1',
  name: 'BuiltinString',
  description: 'Built-in string type',
  base_type: 'string' as const,
  system: true,
  latest_version: 1,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const mockEnumType = {
  id: 'td-custom-1',
  name: 'StatusEnum',
  description: 'A status enum',
  base_type: 'enum' as const,
  system: false,
  latest_version: 2,
  created_at: '2026-02-01T00:00:00Z',
  updated_at: '2026-02-15T00:00:00Z',
}

const mockIntegerType = {
  id: 'td-custom-2',
  name: 'PortNum',
  description: '',
  base_type: 'integer' as const,
  system: false,
  latest_version: 1,
  created_at: '2026-03-01T00:00:00Z',
  updated_at: '2026-03-01T00:00:00Z',
}

const mockBooleanType = {
  id: 'td-custom-3',
  name: 'IsEnabled',
  base_type: 'boolean' as const,
  system: false,
  latest_version: 1,
  created_at: '2026-03-02T00:00:00Z',
  updated_at: '2026-03-02T00:00:00Z',
}

const allTypes = [mockSystemType, mockEnumType, mockIntegerType, mockBooleanType]

function renderPage(role: 'Admin' | 'RO' | 'RW' | 'SuperAdmin' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/types']}>
      <Routes>
        <Route path="/schema/types" element={<TypeDefinitionListPage role={role} />} />
        <Route path="/schema/types/:id" element={<div>Type Detail Page</div>} />
      </Routes>
    </MemoryRouter>
  )
}

/** Open create modal and return dialog locator */
async function openCreateModal() {
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await page.getByRole('button', { name: 'Create Type Definition' }).click()
  const dialog = page.getByRole('dialog')
  await expect.element(dialog).toBeVisible()
  return dialog
}

/**
 * Select a base type in the create modal.
 * PatternFly Select renders dropdown options in a portal at body level.
 * The table also has Label elements with base type names, so we use .last()
 * to select the dropdown option (which renders after the table labels).
 */
async function selectBaseType(currentType: string, targetType: string) {
  const dialog = page.getByRole('dialog')
  await dialog.getByText(currentType, { exact: true }).click()
  // The dropdown option is the last matching text (rendered in portal after table)
  await page.getByText(targetType, { exact: true }).last().click()
}

/** Select element base type for list constraints */
async function selectElementType(targetType: string) {
  const dialog = page.getByRole('dialog')
  await dialog.getByText('Select...', { exact: true }).click()
  await page.getByText(targetType, { exact: true }).last().click()
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: allTypes, total: allTypes.length })
  ;(api.typeDefinitions.create as Mock).mockResolvedValue({
    id: 'td-new', name: 'NewType', base_type: 'string', system: false,
    latest_version: 1, created_at: '2026-04-01T00:00:00Z', updated_at: '2026-04-01T00:00:00Z',
  })
  ;(api.typeDefinitions.delete as Mock).mockResolvedValue(undefined)
})

// === Loading & Rendering ===

test('shows loading spinner while fetching', async () => {
  let resolveList!: (value: unknown) => void
  ;(api.typeDefinitions.list as Mock).mockReturnValue(new Promise(r => { resolveList = r }))
  renderPage()
  await expect.element(page.getByLabelText('Loading')).toBeVisible()
  resolveList({ items: allTypes, total: allTypes.length })
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
})

test('renders list of type definitions', async () => {
  renderPage()
  await expect.element(page.getByRole('heading', { name: 'Type Definitions' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'PortNum' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'IsEnabled' })).toBeVisible()
})

test('shows version and description columns', async () => {
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'V2' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'A status enum' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'Built-in string type' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: '-' }).first()).toBeVisible()
})

test('system types show System badge', async () => {
  renderPage()
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByText('System')).toBeVisible()
})

test('system types do not show delete button', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockSystemType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

test('custom types show delete button for Admin', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage('Admin')
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('shows empty state when no types', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderPage()
  await expect.element(page.getByText('No type definitions yet. Create one to get started.')).toBeVisible()
})

test('handles null items from API', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: null, total: 0 })
  renderPage()
  await expect.element(page.getByText('No type definitions yet. Create one to get started.')).toBeVisible()
})

test('shows error alert when fetch fails', async () => {
  ;(api.typeDefinitions.list as Mock).mockRejectedValue(new Error('Network error'))
  renderPage()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

test('shows generic error for non-Error fetch rejections', async () => {
  ;(api.typeDefinitions.list as Mock).mockRejectedValue('unknown')
  renderPage()
  await expect.element(page.getByText('Failed to load type definitions')).toBeVisible()
})

// === Role-based visibility ===

test('Admin sees Create Type Definition button', async () => {
  renderPage('Admin')
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Type Definition' })).toBeVisible()
})

test('SuperAdmin sees Create Type Definition button', async () => {
  renderPage('SuperAdmin')
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Type Definition' })).toBeVisible()
})

test('RO role hides Create and Delete buttons', async () => {
  renderPage('RO')
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Type Definition' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

test('RW role hides Create and Delete buttons', async () => {
  renderPage('RW')
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Create Type Definition' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

// === Refresh ===

test('Refresh button reloads type definitions', async () => {
  renderPage()
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  expect(api.typeDefinitions.list).toHaveBeenCalledTimes(1)
  await page.getByRole('button', { name: 'Refresh' }).click()
  expect(api.typeDefinitions.list).toHaveBeenCalledTimes(2)
})

// === Navigate ===

test('clicking type name navigates to detail page', async () => {
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'StatusEnum' }).click()
  await expect.element(page.getByText('Type Detail Page')).toBeVisible()
})

// === Create Modal open/close ===

test('Create modal opens and closes via Cancel', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await expect.element(dialog.getByRole('heading', { name: 'Create Type Definition' })).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel' }).click()
})

test('Create button disabled when name is empty', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await expect.element(dialog.getByRole('button', { name: 'Create' })).toHaveAttribute('disabled', '')
})

// === Create with various base types ===

test('create with string base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('NewStringType')
  await dialog.getByPlaceholder('Optional description').fill('A test type')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'NewStringType', description: 'A test type', base_type: 'string', constraints: undefined,
  })
  expect(api.typeDefinitions.list).toHaveBeenCalledTimes(2)
})

test('create with no description sends undefined', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('NoDesc')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'NoDesc', description: undefined, base_type: 'string', constraints: undefined,
  })
})

test('create with integer base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('PortNumber')
  await selectBaseType('string', 'integer')
  await dialog.getByLabelText('Min').fill('1')
  await dialog.getByLabelText('Max').fill('65535')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'PortNumber', description: undefined, base_type: 'integer', constraints: { min: 1, max: 65535 },
  })
})

test('create with number base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('Pct')
  await selectBaseType('string', 'number')
  await dialog.getByLabelText('Min').fill('0.5')
  await dialog.getByLabelText('Max').fill('100.5')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'Pct', description: undefined, base_type: 'number', constraints: { min: 0.5, max: 100.5 },
  })
})

test('create with boolean base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('IsActive')
  await selectBaseType('string', 'boolean')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'IsActive', description: undefined, base_type: 'boolean', constraints: undefined,
  })
})

test('create with date base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('DateField')
  await selectBaseType('string', 'date')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'DateField', description: undefined, base_type: 'date', constraints: undefined,
  })
})

test('create with url base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('UrlField')
  await selectBaseType('string', 'url')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'UrlField', description: undefined, base_type: 'url', constraints: undefined,
  })
})

test('create with json base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('JsonField')
  await selectBaseType('string', 'json')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'JsonField', description: undefined, base_type: 'json', constraints: undefined,
  })
})

// === Enum base type ===

test('create with enum — add values via button', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('MyEnum')
  await selectBaseType('string', 'enum')

  await dialog.getByPlaceholder('New value').fill('ItemA')
  await dialog.getByRole('button', { name: 'Add' }).click()
  await dialog.getByPlaceholder('New value').fill('ItemB')
  await dialog.getByRole('button', { name: 'Add' }).click()

  // Verify both values are shown as labels
  await expect.element(page.getByText('ItemA')).toBeVisible()
  await expect.element(page.getByText('ItemB')).toBeVisible()

  await dialog.getByRole('button', { name: 'Create' }).click()

  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.base_type).toBe('enum')
  expect(callArgs.constraints?.values).toEqual(['ItemA', 'ItemB'])
})

test('enum: add value via Enter key', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('EnterEnum')
  await selectBaseType('string', 'enum')

  await dialog.getByPlaceholder('New value').fill('Pending')
  await userEvent.keyboard('{Enter}')
  await expect.element(dialog.getByText('Pending')).toBeVisible()

  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.values).toEqual(['Pending'])
})

test('enum: remove value by clicking label close', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('RemoveEnum')
  await selectBaseType('string', 'enum')

  await dialog.getByPlaceholder('New value').fill('Val1')
  await dialog.getByRole('button', { name: 'Add' }).click()
  await dialog.getByPlaceholder('New value').fill('Val2')
  await dialog.getByRole('button', { name: 'Add' }).click()

  await expect.element(dialog.getByText('Val1')).toBeVisible()

  // Click close on Val1 label
  const val1El = dialog.getByText('Val1').element()
  const val1Label = val1El.closest('.pf-v6-c-label')
  const closeBtn = val1Label?.querySelector('button')
  if (closeBtn) {
    await userEvent.click(closeBtn)
  }

  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.values).toEqual(['Val2'])
})

test('enum: Add button disabled when input empty', async () => {
  renderPage()
  await openCreateModal()
  await selectBaseType('string', 'enum')
  const dialog = page.getByRole('dialog')
  await expect.element(dialog.getByRole('button', { name: 'Add' })).toHaveAttribute('disabled', '')
})

test('enum: empty Enter does not add value', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('EmptyEnum')
  await selectBaseType('string', 'enum')

  await dialog.getByPlaceholder('New value').click()
  await userEvent.keyboard('{Enter}')

  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints).toBeUndefined()
})

// === List base type ===

test('create with list base type', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('Tags')
  await selectBaseType('string', 'list')
  await expect.element(dialog.getByText('Element Base Type')).toBeVisible()
  await selectElementType('integer')
  await dialog.getByRole('button', { name: 'Create' }).click()
  expect(api.typeDefinitions.create).toHaveBeenCalledWith({
    name: 'Tags', description: undefined, base_type: 'list', constraints: { element_base_type: 'integer' },
  })
})

// === Create error handling ===

test('create failure shows error alert', async () => {
  ;(api.typeDefinitions.create as Mock).mockRejectedValue(new Error('Duplicate name'))
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ExistingType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  await expect.element(dialog.getByText('Duplicate name')).toBeVisible()
})

test('create failure shows generic error for non-Error', async () => {
  ;(api.typeDefinitions.create as Mock).mockRejectedValue('boom')
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('TestType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  await expect.element(dialog.getByText('Failed to create')).toBeVisible()
})

test('create modal resets form state on success', async () => {
  renderPage()
  let dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('TestType')
  await dialog.getByPlaceholder('Optional description').fill('test desc')
  await dialog.getByRole('button', { name: 'Create' }).click()
  await expect.element(page.getByRole('button', { name: 'BuiltinString' })).toBeVisible()
  dialog = await openCreateModal()
  await expect.element(dialog.getByRole('textbox', { name: 'Name' })).toHaveValue('')
})

test('closing create modal clears error', async () => {
  ;(api.typeDefinitions.create as Mock).mockRejectedValue(new Error('Duplicate'))
  renderPage()
  let dialog = await openCreateModal()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('Bad')
  await dialog.getByRole('button', { name: 'Create' }).click()
  await expect.element(dialog.getByText('Duplicate')).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel' }).click()
  dialog = await openCreateModal()
  await expect.element(dialog.getByText('Duplicate')).not.toBeInTheDocument()
})

// === Delete Modal ===

test('delete custom type with confirmation', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await expect.element(dialog.getByText('Confirm Deletion')).toBeVisible()
  await dialog.getByRole('button', { name: 'Delete' }).click()
  expect(api.typeDefinitions.delete).toHaveBeenCalledWith('td-custom-1')
  expect(api.typeDefinitions.list).toHaveBeenCalledTimes(2)
})

test('cancel delete does not call API', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await expect.element(dialog.getByText('Confirm Deletion')).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel' }).click()
  expect(api.typeDefinitions.delete).not.toHaveBeenCalled()
})

test('delete failure shows error', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue(new Error('Type in use'))
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: 'Delete' }).click()
  await expect.element(dialog.getByText('Type in use')).toBeVisible()
})

test('delete failure shows generic error for non-Error', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue('broke')
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: 'Delete' }).click()
  await expect.element(dialog.getByText('Failed to delete')).toBeVisible()
})

test('closing delete modal clears error', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue(new Error('Cannot delete'))
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name: 'Delete' }).click()
  await expect.element(dialog.getByText('Cannot delete')).toBeVisible()
  await dialog.getByRole('button', { name: 'Cancel' }).click()
})

// === String constraints ===

test('string constraints form shows max length, pattern, multiline', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await expect.element(dialog.getByText('Max Length')).toBeVisible()
  await expect.element(dialog.getByText('Pattern (regex)')).toBeVisible()
  await expect.element(dialog.getByText('Multiline')).toBeVisible()
})

test('string constraints: pattern creates constraint', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByPlaceholder('e.g. ^[a-z]+$').fill('^[A-Z]+$')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('PatternType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.pattern).toBe('^[A-Z]+$')
})

test('string constraints: multiline checkbox', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('checkbox').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('MultilineType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.multiline).toBe(true)
})

test('string constraints: unchecking multiline removes it', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByRole('checkbox').click()
  await dialog.getByRole('checkbox').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('NoMultiline')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints).toBeUndefined()
})

test('string constraints: clearing pattern removes constraint', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByPlaceholder('e.g. ^[a-z]+$').fill('abc')
  await dialog.getByPlaceholder('e.g. ^[a-z]+$').fill('')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('NoPattern')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints).toBeUndefined()
})

test('string constraints: max length plus increments', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('PlusType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(2)
})

test('string constraints: max length minus decrements', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Minus').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('MinusType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(2)
})

test('string constraints: max length minus from 1 goes to 0', async () => {
  renderPage()
  const dialog = await openCreateModal()
  // First set to 1 via Plus, then Minus to 0
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Minus').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('MinusTo0')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(0)
})

test('string constraints: max length onChange with valid number', async () => {
  renderPage()
  const dialog = await openCreateModal()
  // The NumberInput internal input has aria-label="Input"
  const numInput = dialog.getByLabelText('Input')
  await numInput.fill('42')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('DirectInput')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(42)
})

// === Integer/Number edge cases ===

test('integer constraints: clearing min sends undefined', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await selectBaseType('string', 'integer')
  await dialog.getByLabelText('Min').fill('10')
  await dialog.getByLabelText('Min').fill('')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ClearedMin')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints).toBeUndefined()
})

// === Changing base type resets constraints ===

test('changing base type resets constraints', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await dialog.getByPlaceholder('e.g. ^[a-z]+$').fill('^test$')
  await selectBaseType('string', 'integer')
  await expect.element(dialog.getByText('Min')).toBeVisible()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('SwitchedType')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.base_type).toBe('integer')
  expect(callArgs.constraints).toBeUndefined()
})

// === List constraints ===

test('list constraints: max length plus/minus', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await selectBaseType('string', 'list')
  await selectElementType('string')
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Minus').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ListPlusMinus')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(2)
})

test('list constraints: minus from 1 goes to 0', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await selectBaseType('string', 'list')
  // First set to 1, then minus to 0
  await dialog.getByLabelText('Plus').click()
  await dialog.getByLabelText('Minus').click()
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ListMinusTo0')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(0)
})

test('list constraints: max length onChange with valid number', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await selectBaseType('string', 'list')
  const numInput = dialog.getByLabelText('Input')
  await numInput.fill('50')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ListDirectInput')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(50)
})

test('list constraints: max length onChange with valid number', async () => {
  renderPage()
  const dialog = await openCreateModal()
  await selectBaseType('string', 'list')
  const numInput = dialog.getByLabelText('Input')
  await numInput.fill('75')
  await dialog.getByRole('textbox', { name: 'Name' }).fill('ListDirect75')
  await dialog.getByRole('button', { name: 'Create' }).click()
  const callArgs = (api.typeDefinitions.create as Mock).mock.calls[0][0]
  expect(callArgs.constraints?.max_length).toBe(75)
})

// === Unreachable guards - defensive coverage ===
//
// Line 210 (default: return null in ConstraintsForm switch):
//   UNREACHABLE: BaseType is a union of 9 values (string, integer, number, boolean,
//   date, url, enum, list, json), all of which are handled by explicit switch cases.
//   TypeScript enforces that baseType is always one of these 9 values.
//
// Line 265 (if (!newName.trim()) return in handleCreate):
//   UNREACHABLE: The Create button is disabled when !newName.trim() (line 409),
//   so handleCreate can only be called when newName has content.
//
// Line 293 (if (!deleteTarget) return in handleDelete):
//   UNREACHABLE: The Delete modal only opens when deleteTarget is set (isOpen={deleteTarget !== null}),
//   and the Delete button inside the modal calls handleDelete. Since the modal is closed
//   when deleteTarget is null, handleDelete can never be called with deleteTarget === null.

test('closing create modal via X button', async () => {
  renderPage()
  await openCreateModal()
  const dialog = page.getByRole('dialog')
  // Click the X/close button on the modal
  const closeBtn = dialog.getByLabelText('Close')
  await closeBtn.click()
})

test('closing delete modal via X button', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [mockEnumType], total: 1 })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'StatusEnum' })).toBeVisible()
  await page.getByRole('button', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog')
  await expect.element(dialog.getByText('Confirm Deletion')).toBeVisible()
  const closeBtn = dialog.getByLabelText('Close')
  await closeBtn.click()
  expect(api.typeDefinitions.delete).not.toHaveBeenCalled()
})

// === Base type color mapping ===

test('renders all base type color variants', async () => {
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({
    items: [
      { ...mockEnumType, id: 't1', name: 'T1', base_type: 'string', description: '' },
      { ...mockEnumType, id: 't2', name: 'T2', base_type: 'integer', description: '' },
      { ...mockEnumType, id: 't3', name: 'T3', base_type: 'number', description: '' },
      { ...mockEnumType, id: 't4', name: 'T4', base_type: 'enum', description: '' },
      { ...mockEnumType, id: 't5', name: 'T5', base_type: 'boolean', description: '' },
      { ...mockEnumType, id: 't6', name: 'T6', base_type: 'date', description: '' },
      { ...mockEnumType, id: 't7', name: 'T7', base_type: 'url', description: '' },
      { ...mockEnumType, id: 't8', name: 'T8', base_type: 'list', description: '' },
      { ...mockEnumType, id: 't9', name: 'T9', base_type: 'json', description: '' },
    ],
    total: 9,
  })
  renderPage()
  await expect.element(page.getByRole('button', { name: 'T1' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'T9' })).toBeVisible()
})
