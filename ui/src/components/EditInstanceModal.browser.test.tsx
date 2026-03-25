import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import EditInstanceModal from './EditInstanceModal'
import type { SnapshotAttribute, EntityInstance } from '../types'

const schemaAttrs: SnapshotAttribute[] = [
  { id: 'sys-name', name: 'name', type: 'string', description: '', ordinal: -2, required: true, system: true },
  { id: 'sys-desc', name: 'description', type: 'string', description: '', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'hostname', type: 'string', description: '', ordinal: 1, required: false },
]

const mockInstance: EntityInstance = {
  id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
  version: 1, attributes: [
    { name: 'hostname', type: 'string', value: 'host-a' },
  ], created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

function renderModal(overrides: Partial<React.ComponentProps<typeof EditInstanceModal>> = {}) {
  const props = {
    instance: mockInstance,
    onClose: vi.fn(),
    schemaAttrs,
    enumValues: {},
    onSubmit: vi.fn().mockResolvedValue(undefined),
    error: null,
    ...overrides,
  }
  return { ...render(<EditInstanceModal {...props} />), props }
}

// T-19.34: Pre-fills form from instance data
test('T-19.34: EditInstanceModal pre-fills form', async () => {
  renderModal()
  // Verify the modal title shows instance name and form is rendered
  await expect.element(page.getByRole('dialog').getByText('Edit inst-a')).toBeVisible()
  // Verify the form is rendered with description label
  await expect.element(page.getByText('Description', { exact: true })).toBeVisible()
})

// T-19.35: Calls onSubmit with updated fields
test('T-19.35: EditInstanceModal calls onSubmit', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Save' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(1, 'inst-a', 'First', expect.objectContaining({ hostname: 'host-a' }))
})

// T-19.36: Shows error when provided
test('T-19.36: EditInstanceModal shows error', async () => {
  renderModal({ error: 'Conflict' })
  await expect.element(page.getByText('Conflict')).toBeVisible()
})

// T-19.37: Closed when instance is null
test('T-19.37: EditInstanceModal closed when null', async () => {
  renderModal({ instance: null })
  // When instance is null, the modal should not be in DOM
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// T-20.27: Pre-fills from initialValues prop
test('T-20.27: EditInstanceModal pre-fills from instance prop', async () => {
  renderModal()
  const nameInput = page.getByLabelText('Name *', { exact: true })
  await expect.element(nameInput).toHaveValue('inst-a')
  const descInput = page.getByRole('dialog').getByRole('textbox', { name: 'Description' })
  await expect.element(descInput).toHaveValue('First')
})

// T-20.28: onSubmit receives updated values
test('T-20.28: EditInstanceModal onSubmit receives updated values', async () => {
  const { props } = renderModal()
  const nameInput = page.getByLabelText('Name *', { exact: true })
  await nameInput.clear()
  await nameInput.fill('new-name')
  await page.getByRole('button', { name: 'Save' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(1, 'new-name', 'First', expect.objectContaining({ hostname: 'host-a' }))
})
