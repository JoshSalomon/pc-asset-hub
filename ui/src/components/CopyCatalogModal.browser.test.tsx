import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import CopyCatalogModal from './CopyCatalogModal'

function renderModal(overrides: Partial<React.ComponentProps<typeof CopyCatalogModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    error: null,
    loading: false,
    ...overrides,
  }
  return { ...render(<CopyCatalogModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.15: Renders name and description inputs
test('T-20.15: CopyCatalogModal renders name and description inputs', async () => {
  renderModal()
  await expect.element(page.getByRole('textbox', { name: /New catalog name/i })).toBeVisible()
  await expect.element(page.getByRole('textbox', { name: /Description/i })).toBeVisible()
})

// T-20.16: Submit disabled when name empty
test('T-20.16: CopyCatalogModal submit disabled when name empty', async () => {
  renderModal()
  const copyBtn = page.getByRole('button', { name: 'Copy' })
  await expect.element(copyBtn).toHaveAttribute('disabled')
})

// T-20.17: Calls onSubmit with name and description
test('T-20.17: CopyCatalogModal calls onSubmit with values', async () => {
  const { props } = renderModal()
  await page.getByRole('textbox', { name: /New catalog name/i }).fill('my-copy')
  await page.getByRole('textbox', { name: /Description/i }).fill('A copy')
  await page.getByRole('button', { name: 'Copy' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('my-copy', 'A copy')
})

// T-20.18: Shows error alert when error prop set
test('T-20.18: CopyCatalogModal shows error', async () => {
  renderModal({ error: 'Name taken' })
  await expect.element(page.getByText('Name taken')).toBeVisible()
})

// T-20.19: Form starts empty when opened
test('T-20.19: CopyCatalogModal starts with empty form', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: /New catalog name/i })
  await expect.element(nameInput).toHaveValue('')
  const descInput = page.getByRole('textbox', { name: /Description/i })
  await expect.element(descInput).toHaveValue('')
})
