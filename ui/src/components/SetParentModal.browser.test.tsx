import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import SetParentModal from './SetParentModal'

function renderModal(overrides: Partial<React.ComponentProps<typeof SetParentModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    instanceName: 'my-instance',
    parentTypeName: 'server',
    parentInstances: [
      { id: 'p1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'server-1', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
      { id: 'p2', entity_type_id: 'et1', catalog_id: 'cat1', name: 'server-2', description: '', version: 1, attributes: [], created_at: '', updated_at: '' },
    ],
    hasParent: false,
    onSubmit: vi.fn().mockResolvedValue(undefined),
    onRemoveParent: vi.fn(),
    error: null,
    ...overrides,
  }
  return { ...render(<SetParentModal {...props} />), props }
}

// T-19.50: Shows parent type (disabled field)
test('T-19.50: SetParentModal shows parent type', async () => {
  renderModal()
  const typeInput = page.getByRole('textbox', { name: 'Container type' })
  await expect.element(typeInput).toHaveValue('server')
})

// T-19.51: Shows parent instances dropdown
test('T-19.51: SetParentModal shows parent instances', async () => {
  renderModal()
  await page.getByText('Select container...').click()
  // Use getByText to find items in the PF Select dropdown
  await expect.element(page.getByText('server-1')).toBeVisible()
  await expect.element(page.getByText('server-2')).toBeVisible()
})

// T-19.52: Calls onSubmit with parentType and parentId
test('T-19.52: SetParentModal calls onSubmit', async () => {
  const { props } = renderModal()
  // Select a parent instance
  await page.getByText('Select container...').click()
  await page.getByText('server-1').click()
  await page.getByRole('button', { name: 'Set Container' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('server', 'p1')
})

// T-19.53: Remove Container button calls onRemoveParent
test('T-19.53: SetParentModal remove calls onRemoveParent', async () => {
  const { props } = renderModal({ hasParent: true })
  await page.getByRole('button', { name: 'Remove Container' }).click()
  expect(props.onRemoveParent).toHaveBeenCalled()
})

// T-19.54: Shows error when provided
test('T-19.54: SetParentModal shows error', async () => {
  renderModal({ error: 'Set parent failed' })
  await expect.element(page.getByText('Set parent failed')).toBeVisible()
})

// T-20.31: onSubmit receives (parentType, parentId)
test('T-20.31: SetParentModal onSubmit receives correct args', async () => {
  const { props } = renderModal()
  await page.getByText('Select container...').click()
  await page.getByText('server-2').click()
  await page.getByRole('button', { name: 'Set Container' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('server', 'p2')
})
