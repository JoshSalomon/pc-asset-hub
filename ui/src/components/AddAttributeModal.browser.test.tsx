import { useState } from 'react'
import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import AddAttributeModal from './AddAttributeModal'
import type { TypeDefinition } from '../types'

const mockTypeDefinitions: TypeDefinition[] = [
  { id: 'td-string', name: 'string', base_type: 'string', system: true, latest_version: 1, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td-number', name: 'number', base_type: 'number', system: true, latest_version: 1, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td1', name: 'Colors', base_type: 'enum', system: false, latest_version: 1, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td2', name: 'Sizes', base_type: 'enum', system: false, latest_version: 1, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof AddAttributeModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    typeDefinitions: mockTypeDefinitions,
    error: null,
    ...overrides,
  }
  return { ...render(<AddAttributeModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.30: Renders modal with title
test('T-20.30: AddAttributeModal renders title', async () => {
  renderModal()
  await expect.element(page.getByText('Add Attribute')).toBeVisible()
})

// T-20.31: Name field is required
test('T-20.31: AddAttributeModal has name field', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toBeVisible()
})

// T-20.32: Add button disabled when name empty
test('T-20.32: AddAttributeModal add disabled when name empty', async () => {
  renderModal()
  const addBtn = page.getByRole('button', { name: 'Add' })
  await expect.element(addBtn).toHaveAttribute('disabled')
})

// T-20.33: Type shows placeholder when no type selected
test('T-20.33: AddAttributeModal type defaults to select placeholder', async () => {
  renderModal()
  await expect.element(page.getByText('Select type...')).toBeVisible()
})

// T-20.34: Shows error
test('T-20.34: AddAttributeModal shows error', async () => {
  renderModal({ error: 'Duplicate name' })
  await expect.element(page.getByText('Duplicate name')).toBeVisible()
})

// T-20.35: Calls onSubmit with form values
test('T-20.35: AddAttributeModal calls onSubmit', async () => {
  const { props } = renderModal()
  // Fill name
  await page.getByRole('textbox', { name: 'Name' }).fill('hostname')
  // Select a type — open the type selector and pick string
  await page.getByText('Select type...').click()
  await page.getByText('string (string)').click()
  // Click Add
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    name: 'hostname',
    typeDefinitionVersionId: 'td-string',
    required: false,
  }))
})

// T-20.36: Cancel calls onClose
test('T-20.36: AddAttributeModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.37: Required checkbox toggles
test('T-20.37: AddAttributeModal required checkbox', async () => {
  const { props } = renderModal()
  await page.getByRole('textbox', { name: 'Name' }).fill('test')
  // Select a type
  await page.getByText('Select type...').click()
  await page.getByText('string (string)').click()
  // Click the Required checkbox
  await page.getByRole('checkbox').click()
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    required: true,
  }))
})

// Line 63: if (!td) return — guard in handleSubmit after find()
// UNREACHABLE: The Add button is disabled when !selectedTdId (line 130),
// so handleSubmit can only run when selectedTdId is set. Since selectedTdId
// was set via the Select dropdown which only shows typeDefinitions items,
// find() always succeeds. This is a defensive guard that cannot be triggered
// through the component's UI.

// T-20.38: Reopen resets form state (I5 — useEffect reset on isOpen)
test('T-20.38: AddAttributeModal reopen resets name field', async () => {
  // Use a wrapper that exposes force close/open functions to simulate the
  // hook's setAddAttrOpen(false) path (isOpen toggled without handleClose)
  let forceClose: () => void = () => {}
  let forceOpen: () => void = () => {}
  function Wrapper() {
    const [open, setOpen] = useState(true)
    forceClose = () => setOpen(false)
    forceOpen = () => setOpen(true)
    return (
      <AddAttributeModal
        isOpen={open}
        onClose={() => setOpen(false)}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        typeDefinitions={mockTypeDefinitions}
        error={null}
      />
    )
  }
  render(<Wrapper />)
  await expect.element(page.getByText('Add Attribute')).toBeVisible()

  // Fill in the name
  await page.getByRole('textbox', { name: 'Name' }).fill('stale-attr')
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('stale-attr')

  // Close by directly setting isOpen=false (simulates hook path)
  forceClose()
  // Wait for the modal to close
  await expect.element(page.getByText('Add Attribute')).not.toBeInTheDocument()

  // Reopen
  forceOpen()
  await expect.element(page.getByText('Add Attribute')).toBeVisible()

  // Name should be reset
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('')
})
