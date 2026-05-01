import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import EditAssociationModal from './EditAssociationModal'
import type { AssociationEditData } from './EditAssociationModal'

const defaultInitialData: AssociationEditData & { sourceName?: string; targetName?: string } = {
  name: 'test-assoc',
  type: 'directional',
  sourceRole: 'src',
  targetRole: 'tgt',
  sourceCardinality: '0..n',
  targetCardinality: '0..n',
}

function renderModal(overrides: Partial<React.ComponentProps<typeof EditAssociationModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSave: vi.fn().mockResolvedValue(undefined),
    initialData: defaultInitialData,
    ...overrides,
  }
  return { ...render(<EditAssociationModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

test('onSave error displays error alert in the modal', async () => {
  const failingSave = vi.fn().mockRejectedValue(new Error('409: association already exists'))
  renderModal({ onSave: failingSave })

  // The name field should be pre-filled
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('test-assoc')

  // Click Save
  await page.getByRole('button', { name: 'Save' }).click()

  // Error alert should appear in the modal
  await expect.element(page.getByText('409: association already exists')).toBeVisible()
})
