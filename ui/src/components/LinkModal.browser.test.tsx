import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import LinkModal from './LinkModal'
import { api } from '../api/client'
import type { SnapshotAssociation, CatalogVersionPin } from '../types'

vi.mock('../api/client', () => ({
  api: {
    instances: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

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

const mockPins: CatalogVersionPin[] = [
  { pin_id: 'pin-1', entity_type_name: 'tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
  { pin_id: 'pin-2', entity_type_name: 'config', entity_type_id: 'et3', entity_type_version_id: 'etv3', version: 1 },
]

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.instances.list as Mock).mockResolvedValue({ items: [], total: 0 })
})

function renderModal(overrides: Partial<React.ComponentProps<typeof LinkModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    catalogName: 'my-catalog',
    pins: mockPins,
    schemaAssocs,
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

// T-19.46: Loads target instances on association selection (internally calls API)
test('T-19.46: LinkModal loads target instances on assoc selection', async () => {
  renderModal()
  await page.getByText('Select association...').click()
  // PF Select renders options as menu items with text content
  const option = page.getByText(/^uses/)
  await expect.element(option).toBeVisible()
  await option.click()
  // Modal now loads data internally — verify API was called
  expect(api.instances.list).toHaveBeenCalledWith('my-catalog', 'tool')
})

// T-19.47: Submit button disabled until assoc and target selected
test('T-19.47: LinkModal submit disabled when fields empty', async () => {
  renderModal()
  const linkBtn = page.getByRole('button', { name: 'Link' })
  await expect.element(linkBtn).toHaveAttribute('disabled')
})

// T-19.48: Calls onSubmit with targetId and assocName
test('T-19.48: LinkModal calls onSubmit', async () => {
  ;(api.instances.list as Mock).mockResolvedValue({
    items: [
      { id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'tool-1', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
    total: 1,
  })
  const { props } = renderModal()
  // Select association (triggers internal load of target instances)
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
  ;(api.instances.list as Mock).mockResolvedValue({
    items: [
      { id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'tool-1', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
    total: 1,
  })
  const { props } = renderModal()
  await page.getByText('Select association...').click()
  await page.getByText(/^uses/).click()
  await page.getByText('Select target instance...').click()
  await page.getByText('tool-1').click()
  await page.getByRole('button', { name: 'Link' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('i2', 'uses')
})

// Line 46: loadLinkTargetInstances guard — catalogName undefined
test('LinkModal guard: loadLinkTargetInstances with undefined catalogName returns early', async () => {
  renderModal({ catalogName: undefined })
  // Open association dropdown and select one
  await page.getByText('Select association...').click()
  await page.getByText(/^uses/).click()
  // api.instances.list should NOT be called because catalogName guard fires
  expect(api.instances.list).not.toHaveBeenCalled()
})

// Line 48: loadLinkTargetInstances guard — assoc not found (no matching outgoing assoc)
// UNREACHABLE: The Select dropdown on line 91 filters to outgoing non-containment
// assocs: `schemaAssocs.filter(a => a.type !== 'containment' && a.direction === 'outgoing')`.
// The find() on line 47 searches by the same criteria (name + outgoing direction).
// Since the user can only select from the filtered list, find() always succeeds.
// This guard cannot be triggered through the component's UI.

// Line 50: loadLinkTargetInstances guard — targetPin not found
test('LinkModal guard: loadLinkTargetInstances with no matching pin returns early', async () => {
  // Provide assocs but NO matching pins, so targetPin = undefined on line 50
  renderModal({ pins: [] })
  await page.getByText('Select association...').click()
  await page.getByText(/^uses/).click()
  // api.instances.list should NOT be called because targetPin guard fires
  expect(api.instances.list).not.toHaveBeenCalled()
})
