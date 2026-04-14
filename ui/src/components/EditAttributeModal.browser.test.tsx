import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import EditAttributeModal from './EditAttributeModal'
import type { TypeDefinition } from '../types'

const mockTypeDefinitions: TypeDefinition[] = [
  { id: 'td-string', name: 'string', base_type: 'string', system: true, latest_version: 1, latest_version_id: 'tdv-string', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td1', name: 'Colors', base_type: 'enum', system: false, latest_version: 1, latest_version_id: 'tdv-colors', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof EditAttributeModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    typeDefinitions: mockTypeDefinitions,
    error: null,
    initialName: 'hostname',
    initialDescription: 'The host',
    initialTypeDefinitionId: '',
    initialRequired: false,
    ...overrides,
  }
  return { ...render(<EditAttributeModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.40: Renders with pre-populated fields
test('T-20.40: EditAttributeModal renders with pre-populated name', async () => {
  renderModal()
  await expect.element(page.getByText('Edit Attribute')).toBeVisible()
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toHaveValue('hostname')
})

// T-20.41: Save button disabled when name empty
test('T-20.41: EditAttributeModal save disabled when name empty', async () => {
  renderModal({ initialName: '' })
  const saveBtn = page.getByRole('button', { name: 'Save' })
  await expect.element(saveBtn).toHaveAttribute('disabled')
})

// T-20.42: Calls onSubmit with updated values
test('T-20.42: EditAttributeModal calls onSubmit', async () => {
  const { props } = renderModal({ initialTypeDefinitionId: 'td-string' })
  // Change the name
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await nameInput.clear()
  await nameInput.fill('hostname2')
  await page.getByRole('button', { name: 'Save' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    name: 'hostname2',
    typeDefinitionVersionId: 'tdv-string',
  }))
})

// T-20.43: Shows error
test('T-20.43: EditAttributeModal shows error', async () => {
  renderModal({ error: 'Invalid type' })
  await expect.element(page.getByText('Invalid type')).toBeVisible()
})

// T-20.44: Cancel calls onClose
test('T-20.44: EditAttributeModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.45: Save with unresolvable type definition does not call onSubmit
test('T-20.45: EditAttributeModal save with unknown type skips onSubmit', async () => {
  const { props } = renderModal({ initialTypeDefinitionId: 'nonexistent-td' })
  await page.getByRole('button', { name: 'Save' }).click()
  expect(props.onSubmit).not.toHaveBeenCalled()
})
