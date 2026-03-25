import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import ReplaceCatalogModal from './ReplaceCatalogModal'
import type { Catalog } from '../types'

const availableCatalogs: Catalog[] = [
  { id: 'c1', name: 'prod-catalog', description: '', catalog_version_id: 'cv1', validation_status: 'valid', published: true, created_at: '', updated_at: '' },
  { id: 'c2', name: 'stage-catalog', description: '', catalog_version_id: 'cv2', validation_status: 'valid', published: false, created_at: '', updated_at: '' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof ReplaceCatalogModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    availableCatalogs,
    error: null,
    loading: false,
    ...overrides,
  }
  return { ...render(<ReplaceCatalogModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.20: Renders target catalog dropdown
test('T-20.20: ReplaceCatalogModal renders target dropdown', async () => {
  renderModal()
  await expect.element(page.getByText('Select target catalog...')).toBeVisible()
})

// T-20.21: Submit disabled when target not selected
test('T-20.21: ReplaceCatalogModal submit disabled when no target', async () => {
  renderModal()
  const replaceBtn = page.getByRole('button', { name: 'Replace' })
  await expect.element(replaceBtn).toHaveAttribute('disabled')
})

// T-20.22: Calls onSubmit with target and archiveName
test('T-20.22: ReplaceCatalogModal calls onSubmit', async () => {
  const { props } = renderModal()
  // Select target
  await page.getByText('Select target catalog...').click()
  await page.getByText('prod-catalog').click()
  // Set archive name
  await page.getByRole('textbox', { name: /Archive name/i }).fill('prod-archive')
  await page.getByRole('button', { name: 'Replace' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('prod-catalog', 'prod-archive')
})

// T-20.23: Shows error alert when error prop set
test('T-20.23: ReplaceCatalogModal shows error', async () => {
  renderModal({ error: 'Replace failed' })
  await expect.element(page.getByText('Replace failed')).toBeVisible()
})

// T-20.24: Archive name input is optional
test('T-20.24: ReplaceCatalogModal archive name optional', async () => {
  const { props } = renderModal()
  await page.getByText('Select target catalog...').click()
  await page.getByText('stage-catalog').click()
  // Submit without archive name
  await page.getByRole('button', { name: 'Replace' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith('stage-catalog', '')
})
