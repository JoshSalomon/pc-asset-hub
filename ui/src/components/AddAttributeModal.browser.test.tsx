import { useState } from 'react'
import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import AddAttributeModal from './AddAttributeModal'
import type { Enum } from '../types'

const mockEnums: Enum[] = [
  { id: 'enum1', name: 'Colors', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'enum2', name: 'Sizes', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof AddAttributeModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    enums: mockEnums,
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

// T-20.33: Type defaults to string
test('T-20.33: AddAttributeModal type defaults to string', async () => {
  renderModal()
  await expect.element(page.getByText('string')).toBeVisible()
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
  // Click Add
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    name: 'hostname',
    type: 'string',
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
  // Click the Required checkbox
  await page.getByRole('checkbox').click()
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    required: true,
  }))
})

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
        enums={mockEnums}
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
