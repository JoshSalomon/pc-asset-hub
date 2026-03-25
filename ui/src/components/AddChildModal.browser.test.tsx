import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import AddChildModal from './AddChildModal'
import type { SnapshotAssociation } from '../types'

const containmentAssocs: SnapshotAssociation[] = [
  {
    id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing',
    target_entity_type_id: 'et2', target_entity_type_name: 'tool',
    source_role: '', target_role: '', source_cardinality: '', target_cardinality: '',
  },
  {
    id: 'assoc2', name: 'configs', type: 'containment', direction: 'outgoing',
    target_entity_type_id: 'et3', target_entity_type_name: 'config',
    source_role: '', target_role: '', source_cardinality: '', target_cardinality: '',
  },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof AddChildModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    schemaAssocs: containmentAssocs,
    childSchemaAttrs: [],
    childEnumValues: {},
    availableInstances: [],
    onChildTypeChange: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    error: null,
    ...overrides,
  }
  return { ...render(<AddChildModal {...props} />), props }
}

// T-19.38: Shows child type selector from containment assocs
test('T-19.38: AddChildModal shows child type selector', async () => {
  renderModal()
  await expect.element(page.getByText('Select child type...')).toBeVisible()
})

// T-19.39: Loads child schema on type selection (via onChildTypeChange callback)
test('T-19.39: AddChildModal calls onChildTypeChange on type selection', async () => {
  const { props } = renderModal()
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  expect(props.onChildTypeChange).toHaveBeenCalledWith('tool')
})

// T-19.40: Create mode shows name, description fields
test('T-19.40: AddChildModal create mode shows form fields', async () => {
  renderModal()
  // Select a child type first
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  await expect.element(page.getByRole('textbox', { name: /Name/i })).toBeVisible()
})

// T-19.41: Adopt mode shows instance selector
test('T-19.41: AddChildModal adopt mode shows instance selector', async () => {
  renderModal({
    availableInstances: [
      { id: 'i1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'orphan-tool', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
  })
  // Select a type first
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  // Switch to adopt mode
  await page.getByText('Create New').click()
  await page.getByText('Adopt Existing').click()
  await expect.element(page.getByText('Select instance...')).toBeVisible()
})

// T-19.42: Calls onSubmit with create data (childTypeName + newChildName must be set)
test('T-19.42: AddChildModal calls onSubmit on create', async () => {
  const { props } = renderModal()
  // Select type
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  // Fill name
  await page.getByRole('textbox', { name: /Name/i }).fill('new-tool')
  const createBtn = page.getByRole('button', { name: 'Create', exact: true })
  await expect.element(createBtn).not.toHaveAttribute('disabled')
  await createBtn.click()
  expect(props.onSubmit).toHaveBeenCalledWith('tool', 'create', expect.objectContaining({ name: 'new-tool' }))
})

// T-19.43: Calls onSubmit with adopt data
test('T-19.43: AddChildModal calls onSubmit on adopt', async () => {
  const { props } = renderModal({
    availableInstances: [
      { id: 'i1', entity_type_id: 'et2', catalog_id: 'cat1', name: 'orphan-tool', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
  })
  // Select type
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  // Switch to adopt mode
  await page.getByText('Create New').click()
  await page.getByText('Adopt Existing').click()
  // Select instance
  await page.getByText('Select instance...').click()
  await page.getByText('orphan-tool').click()
  const adoptBtn = page.getByRole('button', { name: 'Adopt', exact: true })
  await expect.element(adoptBtn).not.toHaveAttribute('disabled')
  await adoptBtn.click()
  expect(props.onSubmit).toHaveBeenCalledWith('tool', 'adopt', expect.objectContaining({ adoptInstanceId: 'i1' }))
})

// T-19.44: Shows error when provided
test('T-19.44: AddChildModal shows error', async () => {
  renderModal({ error: 'Failed to add child' })
  await expect.element(page.getByText('Failed to add child')).toBeVisible()
})

// T-20.29: onSubmit receives (childType, mode, data)
test('T-20.29: AddChildModal onSubmit receives typed args', async () => {
  const { props } = renderModal()
  await page.getByText('Select child type...').click()
  await page.getByText('tool').click()
  await page.getByRole('textbox', { name: /Name/i }).fill('my-tool')
  await page.getByRole('button', { name: 'Create', exact: true }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('tool', 'create', expect.objectContaining({
    name: 'my-tool',
  }))
})
