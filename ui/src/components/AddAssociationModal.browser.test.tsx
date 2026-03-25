import { useState } from 'react'
import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import AddAssociationModal from './AddAssociationModal'
import type { EntityType } from '../types'

const mockEntityTypes: EntityType[] = [
  { id: 'et-1', name: 'MLModel', created_at: '', updated_at: '' },
  { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
  { id: 'et-3', name: 'Dataset', created_at: '', updated_at: '' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof AddAssociationModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    entityTypes: mockEntityTypes,
    currentEntityTypeId: 'et-1',
    error: null,
    ...overrides,
  }
  return { ...render(<AddAssociationModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.50: Renders modal with title
test('T-20.50: AddAssociationModal renders title', async () => {
  renderModal()
  await expect.element(page.getByText('Add Association')).toBeVisible()
})

// T-20.51: Has name field
test('T-20.51: AddAssociationModal has name field', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toBeVisible()
})

// T-20.52: Add button disabled when target not selected or name empty
test('T-20.52: AddAssociationModal add disabled when incomplete', async () => {
  renderModal()
  const addBtn = page.getByRole('button', { name: 'Add' })
  await expect.element(addBtn).toHaveAttribute('disabled')
})

// T-20.53: Shows error
test('T-20.53: AddAssociationModal shows error', async () => {
  renderModal({ error: 'Already exists' })
  await expect.element(page.getByText('Already exists')).toBeVisible()
})

// T-20.54: Cancel calls onClose
test('T-20.54: AddAssociationModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.55: Type defaults to containment
test('T-20.55: AddAssociationModal type defaults to containment', async () => {
  renderModal()
  await expect.element(page.getByText('containment')).toBeVisible()
})

// T-20.56: Reopen resets form state (I5 — useEffect reset on isOpen)
test('T-20.56: AddAssociationModal reopen resets name field', async () => {
  // Use a wrapper that exposes a "force close" button that sets isOpen=false
  // without calling handleClose, simulating the hook's setAddAssocOpen(false) path
  let forceClose: () => void = () => {}
  let forceOpen: () => void = () => {}
  function Wrapper() {
    const [open, setOpen] = useState(true)
    forceClose = () => setOpen(false)
    forceOpen = () => setOpen(true)
    return (
      <AddAssociationModal
        isOpen={open}
        onClose={() => setOpen(false)}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        entityTypes={mockEntityTypes}
        currentEntityTypeId="et-1"
        error={null}
      />
    )
  }
  render(<Wrapper />)
  await expect.element(page.getByText('Add Association')).toBeVisible()

  // Fill in the name
  await page.getByRole('textbox', { name: 'Name' }).fill('stale-name')
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('stale-name')

  // Close by directly setting isOpen=false (simulates hook path)
  forceClose()
  // Wait for the modal to close
  await expect.element(page.getByText('Add Association')).not.toBeInTheDocument()

  // Reopen
  forceOpen()
  await expect.element(page.getByText('Add Association')).toBeVisible()

  // Name should be reset
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('')
})
