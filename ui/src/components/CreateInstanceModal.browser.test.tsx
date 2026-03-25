import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import CreateInstanceModal from './CreateInstanceModal'
import type { SnapshotAttribute } from '../types'

const schemaAttrs: SnapshotAttribute[] = [
  { id: 'sys-name', name: 'name', type: 'string', description: '', ordinal: -2, required: true, system: true },
  { id: 'sys-desc', name: 'description', type: 'string', description: '', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'color', type: 'enum', description: '', ordinal: 1, required: false, enum_id: 'enum1' },
  { id: 'a2', name: 'port', type: 'number', description: '', ordinal: 2, required: true },
]

const enumValues = { enum1: ['red', 'green', 'blue'] }

function renderModal(overrides: Partial<React.ComponentProps<typeof CreateInstanceModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    entityTypeName: 'model',
    schemaAttrs,
    enumValues,
    onSubmit: vi.fn().mockResolvedValue(undefined),
    error: null,
    ...overrides,
  }
  return { ...render(<CreateInstanceModal {...props} />), props }
}

// T-19.27: Renders system attrs (Name required, Description optional)
test('T-19.27: CreateInstanceModal renders system attrs', async () => {
  renderModal()
  await expect.element(page.getByText('Name')).toBeVisible()
  await expect.element(page.getByText('Description')).toBeVisible()
  // Name field should have required indicator
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toBeVisible()
})

// T-19.28: Renders custom attrs from schemaAttrs
test('T-19.28: CreateInstanceModal renders custom attrs', async () => {
  renderModal()
  await expect.element(page.getByText('port *')).toBeVisible()
  await expect.element(page.getByText('color')).toBeVisible()
})

// T-19.29: Renders enum select for enum attributes
test('T-19.29: CreateInstanceModal renders enum select', async () => {
  renderModal()
  // The enum attr 'color' should show a Select (MenuToggle with 'Select...')
  await expect.element(page.getByText('Select...')).toBeVisible()
})

// T-19.30: Submit button disabled when name empty
test('T-19.30: CreateInstanceModal create disabled when name empty', async () => {
  renderModal()
  const createBtn = page.getByRole('button', { name: 'Create' })
  await expect.element(createBtn).toBeVisible()
  // Button should be disabled
  await expect.element(createBtn).toHaveAttribute('disabled')
})

// T-19.31: Calls onSubmit with correct args on submit
test('T-19.31: CreateInstanceModal calls onSubmit', async () => {
  const { props } = renderModal()
  // Fill in the name field
  await page.getByRole('textbox', { name: 'Name' }).fill('my-instance')
  await page.getByRole('button', { name: 'Create' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('my-instance', '', expect.any(Object))
})

// T-19.32: Shows error when provided
test('T-19.32: CreateInstanceModal shows error', async () => {
  renderModal({ error: 'Duplicate name' })
  await expect.element(page.getByText('Duplicate name')).toBeVisible()
})

// T-19.33: onClose called on cancel
test('T-19.33: CreateInstanceModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.25: fills form and onSubmit receives (name, desc, attrs)
test('T-20.25: CreateInstanceModal fills form and onSubmit receives typed values', async () => {
  const { props } = renderModal()
  await page.getByRole('textbox', { name: 'Name' }).fill('test-inst')
  await page.getByRole('textbox', { name: 'Description' }).fill('A description')
  await page.getByRole('button', { name: 'Create' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('test-inst', 'A description', expect.any(Object))
})

// Coverage: submitting with empty custom attr skips it in buildTypedAttrs
test('CreateInstanceModal skips empty custom attr in onSubmit', async () => {
  const { props } = renderModal()
  await page.getByRole('textbox', { name: 'Name' }).fill('my-inst')
  // Leave hostname empty (default) — buildTypedAttrs should skip it
  await page.getByRole('button', { name: 'Create' }).click()
  const call = props.onSubmit.mock.calls[0]
  // attrs should NOT contain hostname since it's empty
  expect(call[2]).toEqual({})
})

// T-20.26: form starts empty when opened (reset on open is via useEffect on isOpen)
test('T-20.26: CreateInstanceModal starts with empty form', async () => {
  renderModal()
  // When opened fresh, fields should be empty
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toHaveValue('')
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await expect.element(descInput).toHaveValue('')
})
