import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import CopyAttributesModal from './CopyAttributesModal'
import type { EntityType, Attribute } from '../types'

const mockEntityTypes: EntityType[] = [
  { id: 'et-1', name: 'MLModel', created_at: '', updated_at: '' },
  { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
]

const mockSourceAttributes: Attribute[] = [
  { id: 'sa1', name: 'color', description: 'Color', base_type: 'enum', ordinal: 0, required: false },
  { id: 'sa2', name: 'hostname', description: 'Host', base_type: 'string', ordinal: 1, required: false },
  { id: 'sa3', name: 'count', description: 'Count', base_type: 'integer', ordinal: 2, required: false },
  { id: 'sa4', name: 'active', description: 'Active', base_type: 'boolean', ordinal: 3, required: false },
]

const existingAttributes: Attribute[] = [
  { id: 'a1', name: 'hostname', description: 'The host', base_type: 'string', ordinal: 0, required: false },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof CopyAttributesModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    onLoadSource: vi.fn().mockResolvedValue(undefined),
    entityTypes: mockEntityTypes,
    currentEntityTypeId: 'et-1',
    sourceAttributes: [] as Attribute[],
    existingAttributes: existingAttributes,
    error: null,
    ...overrides,
  }
  return { ...render(<CopyAttributesModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.60: Renders modal with title
test('T-20.60: CopyAttributesModal renders title', async () => {
  renderModal()
  await expect.element(page.getByText('Copy Attributes from Another Type')).toBeVisible()
})

// T-20.61: Copy Selected button disabled when no attrs selected
test('T-20.61: CopyAttributesModal copy disabled when nothing selected', async () => {
  renderModal()
  const copyBtn = page.getByRole('button', { name: 'Copy Selected' })
  await expect.element(copyBtn).toHaveAttribute('disabled')
})

// T-20.62: Shows source attribute table when provided
test('T-20.62: CopyAttributesModal shows source attrs table', async () => {
  renderModal({ sourceAttributes: mockSourceAttributes })
  await expect.element(page.getByRole('gridcell', { name: 'color', exact: true })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'hostname', exact: true })).toBeVisible()
})

// T-20.63: Conflict detection marks existing attrs
test('T-20.63: CopyAttributesModal shows conflict status', async () => {
  renderModal({ sourceAttributes: mockSourceAttributes })
  // hostname exists in existingAttributes, so should show Conflict
  await expect.element(page.getByText('Conflict')).toBeVisible()
  // color, count, active don't exist, so should show Available (3 instances)
  await expect.element(page.getByText('Available').first()).toBeVisible()
})

// T-20.64: Cancel calls onClose
test('T-20.64: CopyAttributesModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.65: Shows error
test('T-20.65: CopyAttributesModal shows error', async () => {
  renderModal({ error: 'Copy failed' })
  await expect.element(page.getByText('Copy failed')).toBeVisible()
})

// T-20.34: onSubmit receives selected attr names
test('T-20.34: CopyAttributesModal onSubmit receives selected attrs', async () => {
  const { props } = renderModal({ sourceAttributes: mockSourceAttributes })
  // Select the non-conflicting 'color' checkbox
  const checkboxes = page.getByRole('checkbox')
  // First checkbox (color) should be enabled, second (hostname) should be disabled
  await checkboxes.first().click()
  await page.getByRole('button', { name: 'Copy Selected' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(['color'])
})
