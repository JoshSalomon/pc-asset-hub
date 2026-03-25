import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import LinkModal from './LinkModal'
import type { SnapshotAssociation } from '../types'

const schemaAssocs: SnapshotAssociation[] = [
  {
    id: 'assoc1', name: 'uses', type: 'directional', direction: 'outgoing',
    target_entity_type_id: 'et2', target_entity_type_name: 'tool',
    source_role: '', target_role: '', source_cardinality: '', target_cardinality: '',
  },
  {
    id: 'assoc2', name: 'tools', type: 'containment', direction: 'outgoing',
    target_entity_type_id: 'et3', target_entity_type_name: 'config',
    source_role: '', target_role: '', source_cardinality: '', target_cardinality: '',
  },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof LinkModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    schemaAssocs,
    linkTargetInstances: [],
    onAssocChange: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    error: null,
    ...overrides,
  }
  return { ...render(<LinkModal {...props} />), props }
}

// T-19.45: Shows association selector from outgoing non-containment assocs
test('T-19.45: LinkModal shows association selector', async () => {
  renderModal()
  await expect.element(page.getByText('Select association...')).toBeVisible()
})

// T-19.46: Loads target instances on association selection (via onAssocChange)
test('T-19.46: LinkModal calls onAssocChange on assoc selection', async () => {
  const { props } = renderModal()
  await page.getByText('Select association...').click()
  // PF Select renders options as menu items with text content
  const option = page.getByText(/^uses/)
  await expect.element(option).toBeVisible()
  await option.click()
  expect(props.onAssocChange).toHaveBeenCalledWith('uses')
})

// T-19.47: Submit button disabled until assoc and target selected
test('T-19.47: LinkModal submit disabled when fields empty', async () => {
  renderModal()
  const linkBtn = page.getByRole('button', { name: 'Link' })
  await expect.element(linkBtn).toHaveAttribute('disabled')
})

// T-19.48: Calls onSubmit with targetId and assocName
test('T-19.48: LinkModal calls onSubmit', async () => {
  const { props } = renderModal({
    linkTargetInstances: [
      { id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'tool-1', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
  })
  // Select association
  await page.getByText('Select association...').click()
  await page.getByText(/^uses/).click()
  // Select target
  await page.getByText('Select target instance...').click()
  await page.getByText('tool-1').click()
  // Submit
  await page.getByRole('button', { name: 'Link' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.any(String), 'uses')
})

// T-19.49: Shows error when provided
test('T-19.49: LinkModal shows error', async () => {
  renderModal({ error: 'Link failed' })
  await expect.element(page.getByText('Link failed')).toBeVisible()
})

// T-20.30: LinkModal onSubmit receives (targetId, assocName)
test('T-20.30: LinkModal onSubmit receives correct args', async () => {
  const { props } = renderModal({
    linkTargetInstances: [
      { id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'tool-1', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
  })
  await page.getByText('Select association...').click()
  await page.getByText(/^uses/).click()
  await page.getByText('Select target instance...').click()
  await page.getByText('tool-1').click()
  await page.getByRole('button', { name: 'Link' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('i2', 'uses')
})
