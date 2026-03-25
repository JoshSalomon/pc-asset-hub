import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import RenameEntityTypeModal from './RenameEntityTypeModal'

function renderModal(overrides: Partial<React.ComponentProps<typeof RenameEntityTypeModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    currentName: 'MLModel',
    error: null,
    deepCopyWarningOpen: false,
    pendingNewName: '',
    onDeepCopyConfirm: vi.fn().mockResolvedValue(undefined),
    onDeepCopyCancel: vi.fn(),
    ...overrides,
  }
  return { ...render(<RenameEntityTypeModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.70: Renders rename modal
test('T-20.70: RenameEntityTypeModal renders title', async () => {
  renderModal()
  await expect.element(page.getByText('Rename Entity Type')).toBeVisible()
})

// T-20.71: Pre-populates with current name
test('T-20.71: RenameEntityTypeModal pre-populates name', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: 'New Name' })
  await expect.element(nameInput).toHaveValue('MLModel')
})

// T-20.72: Rename disabled when name unchanged
test('T-20.72: RenameEntityTypeModal rename disabled when unchanged', async () => {
  renderModal()
  const renameBtn = page.getByRole('button', { name: 'Rename' })
  await expect.element(renameBtn).toHaveAttribute('disabled')
})

// T-20.73: Rename enabled when name changed
test('T-20.73: RenameEntityTypeModal rename enabled when changed', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: 'New Name' })
  await nameInput.clear()
  await nameInput.fill('NewModel')
  const renameBtn = page.getByRole('button', { name: 'Rename' })
  await expect.element(renameBtn).not.toHaveAttribute('disabled')
})

// T-20.74: Calls onSubmit with new name
test('T-20.74: RenameEntityTypeModal calls onSubmit', async () => {
  const { props } = renderModal()
  const nameInput = page.getByRole('textbox', { name: 'New Name' })
  await nameInput.clear()
  await nameInput.fill('NewModel')
  await page.getByRole('button', { name: 'Rename' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('NewModel', false)
})

// T-20.75: Shows error
test('T-20.75: RenameEntityTypeModal shows error', async () => {
  renderModal({ error: 'Name taken' })
  await expect.element(page.getByText('Name taken')).toBeVisible()
})

// T-20.76: Deep copy warning modal renders
test('T-20.76: RenameEntityTypeModal deep copy warning', async () => {
  renderModal({ isOpen: false, deepCopyWarningOpen: true, pendingNewName: 'NewModel' })
  await expect.element(page.getByText('Deep Copy Required')).toBeVisible()
  await expect.element(page.getByText(/NewModel/)).toBeVisible()
})

// T-20.77: Deep copy confirm calls callback
test('T-20.77: RenameEntityTypeModal deep copy confirm', async () => {
  const { props } = renderModal({ isOpen: false, deepCopyWarningOpen: true, pendingNewName: 'NewModel' })
  await page.getByRole('button', { name: 'Create Copy' }).click()
  expect(props.onDeepCopyConfirm).toHaveBeenCalled()
})

// T-20.78: Cancel calls onClose
test('T-20.78: RenameEntityTypeModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})
