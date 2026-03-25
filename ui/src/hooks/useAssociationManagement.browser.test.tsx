import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useAssociationManagement } from './useAssociationManagement'

vi.mock('../api/client', () => ({
  api: {
    associations: { create: vi.fn(), delete: vi.fn(), edit: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

function TestComponent({ entityTypeId }: { entityTypeId: string }) {
  const onRefresh = vi.fn()
  const setError = vi.fn()
  const mgmt = useAssociationManagement({ entityTypeId, onRefresh, setError })

  return (
    <div>
      <span data-testid="add-open">{String(mgmt.addAssocOpen)}</span>
      <span data-testid="edit-open">{String(mgmt.editAssocOpen)}</span>
      <span data-testid="edit-data-name">{mgmt.editAssocData.name}</span>
      <span data-testid="edit-assoc-error">{mgmt.editAssocError || ''}</span>
      <button onClick={() => mgmt.setAddAssocOpen(true)}>Open Add</button>
      <button onClick={() => mgmt.setAddAssocOpen(false)}>Close Add</button>
      <button onClick={() => mgmt.handleAddAssociation({
        name: 'tools',
        targetId: 'et-2',
        type: 'containment',
        sourceRole: 'parent',
        targetRole: 'child',
        sourceCardinality: '0..1',
        targetCardinality: '0..n',
        sourceCardCustom: false,
        targetCardCustom: false,
        sourceCardMin: '',
        sourceCardMax: '',
        targetCardMin: '',
        targetCardMax: '',
      })}>Add Assoc</button>
      <button onClick={() => mgmt.handleDeleteAssociation('tools')}>Delete Assoc</button>
      <button onClick={() => mgmt.openEditAssoc({
        id: 'assoc1', entity_type_version_id: 'v1', name: 'tools',
        target_entity_type_id: 'et-2', type: 'containment',
        source_role: 'parent', target_role: 'child',
        source_cardinality: '0..1', target_cardinality: '0..n',
        direction: 'outgoing',
      })}>Open Edit</button>
      <button onClick={() => mgmt.handleEditAssociationSave({
        name: 'tools-renamed', type: 'directional',
        sourceRole: '', targetRole: '',
        sourceCardinality: '0..n', targetCardinality: '0..n',
      }).catch(() => { /* re-thrown error handled by caller (e.g. modal) */ })}>Save Edit</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.20: Add association modal toggles
test('T-20.20: useAssociationManagement add modal toggles', async () => {
  render(<TestComponent entityTypeId="et-1" />)
  await expect.element(page.getByTestId('add-open')).toHaveTextContent('false')
  await page.getByRole('button', { name: 'Open Add' }).click()
  await expect.element(page.getByTestId('add-open')).toHaveTextContent('true')
})

// T-20.21: handleAddAssociation calls API
test('T-20.21: useAssociationManagement add calls api', async () => {
  ;(api.associations.create as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" />)
  await page.getByRole('button', { name: 'Add Assoc' }).click()
  expect(api.associations.create).toHaveBeenCalledWith('et-1', expect.objectContaining({
    target_entity_type_id: 'et-2',
    type: 'containment',
    name: 'tools',
  }))
})

// T-20.22: handleDeleteAssociation calls API
test('T-20.22: useAssociationManagement delete calls api', async () => {
  ;(api.associations.delete as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" />)
  await page.getByRole('button', { name: 'Delete Assoc' }).click()
  expect(api.associations.delete).toHaveBeenCalledWith('et-1', 'tools')
})

// T-20.23: openEditAssoc populates edit data
test('T-20.23: useAssociationManagement edit opens with data', async () => {
  render(<TestComponent entityTypeId="et-1" />)
  await page.getByRole('button', { name: 'Open Edit' }).click()
  await expect.element(page.getByTestId('edit-open')).toHaveTextContent('true')
  await expect.element(page.getByTestId('edit-data-name')).toHaveTextContent('tools')
})

// T-20.24: handleEditAssociationSave calls API
test('T-20.24: useAssociationManagement edit save calls api', async () => {
  ;(api.associations.edit as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" />)
  // Open edit first to set editAssocData
  await page.getByRole('button', { name: 'Open Edit' }).click()
  await page.getByRole('button', { name: 'Save Edit' }).click()
  expect(api.associations.edit).toHaveBeenCalledWith('et-1', 'tools', expect.objectContaining({
    name: 'tools-renamed',
    type: 'directional',
  }))
})

// T-20.25: handleEditAssociationSave error is caught and re-thrown
// The hook sets editAssocError AND re-throws so callers (e.g. EditAssociationModal)
// can also catch and display the error in their own UI.
test('T-20.25: useAssociationManagement edit save catches errors', async () => {
  ;(api.associations.edit as Mock).mockRejectedValue(new Error('edit failed'))

  render(<TestComponent entityTypeId="et-1" />)
  // Open edit first to set editAssocData
  await page.getByRole('button', { name: 'Open Edit' }).click()
  await page.getByRole('button', { name: 'Save Edit' }).click()

  // The edit modal should still be open (not closed on error)
  await expect.element(page.getByTestId('edit-open')).toHaveTextContent('true')
  // The error should be surfaced in hook state
  await expect.element(page.getByTestId('edit-assoc-error')).toHaveTextContent('edit failed')
})
