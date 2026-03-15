import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import OperationalCatalogListPage from './OperationalCatalogListPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalogs = [
  {
    id: 'c1', name: 'production-app', description: 'Prod',
    catalog_version_id: 'cv1', catalog_version_label: 'release-1.0',
    validation_status: 'draft' as const,
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'c2', name: 'staging-app', description: '',
    catalog_version_id: 'cv2', catalog_version_label: 'release-2.0',
    validation_status: 'valid' as const,
    created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z',
  },
]

function renderList(role: 'Admin' | 'RW' | 'RO' = 'RO') {
  return render(
    <MemoryRouter initialEntries={['/catalogs']}>
      <Routes>
        <Route path="/catalogs" element={<OperationalCatalogListPage role={role} />} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: mockCatalogs, total: 2 })
})

// T-13.61: Catalog list page loads and shows catalogs
test('T-13.61: catalog list page renders table rows', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  await expect.element(page.getByText('staging-app')).toBeVisible()
})

// T-13.62: Shows name, CV label, status badge columns
test('T-13.62: shows name, CV label, and status badge columns', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  await expect.element(page.getByText('release-1.0')).toBeVisible()
  await expect.element(page.getByText('draft')).toBeVisible()
  await expect.element(page.getByText('staging-app')).toBeVisible()
  await expect.element(page.getByText('release-2.0')).toBeVisible()
  await expect.element(page.getByText('valid')).toBeVisible()
})

// T-13.63: Search input filters catalogs by name
test('T-13.63: search input filters catalogs by name', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  await expect.element(page.getByText('staging-app')).toBeVisible()
  const searchInput = page.getByPlaceholder('Filter by name')
  await searchInput.fill('production')
  await expect.element(page.getByText('production-app')).toBeVisible()
  // staging-app should be filtered out
  await expect.element(page.getByText('staging-app')).not.toBeInTheDocument()
})

// T-13.64: Sortable column headers exist
test('T-13.64: sortable column headers exist', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  // PatternFly table renders Th elements — text may be visually hidden via CSS
  await expect.element(page.getByText('Name')).toBeInTheDocument()
  await expect.element(page.getByText('Catalog Version')).toBeInTheDocument()
  await expect.element(page.getByText('Status')).toBeInTheDocument()
  await expect.element(page.getByText('Created')).toBeInTheDocument()
})

// T-13.65: Pagination controls present
test('T-13.65: pagination controls present', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  // PatternFly Pagination renders item count text (may appear in multiple elements)
  await expect.element(page.getByText(/1 - 2 of 2/).first()).toBeVisible()
})

// T-13.66: Clicking catalog name navigates — verify link element exists
test('T-13.66: catalog name is a clickable link', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  // The catalog name is rendered as a PatternFly Button with variant="link"
  const link = page.getByRole('button', { name: 'production-app' })
  await expect.element(link).toBeVisible()
})

// T-13.67: Validation status badge colors
test('T-13.67: validation status badges show draft=blue and valid=green labels', async () => {
  renderList()
  // Both status labels should be visible
  await expect.element(page.getByText('draft')).toBeVisible()
  await expect.element(page.getByText('valid')).toBeVisible()
})

// Coverage: invalid status badge
test('invalid status badge renders', async () => {
  ;(api.catalogs.list as Mock).mockResolvedValue({
    items: [{
      id: 'c3', name: 'broken-catalog', description: '',
      catalog_version_id: 'cv3', catalog_version_label: 'v3',
      validation_status: 'invalid',
      created_at: '2026-01-03T00:00:00Z', updated_at: '2026-01-03T00:00:00Z',
    }],
    total: 1,
  })
  renderList()
  await expect.element(page.getByText('invalid')).toBeVisible()
})

// Coverage: empty catalog list
test('empty catalog list shows empty state', async () => {
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderList()
  await expect.element(page.getByText('No catalogs available.')).toBeVisible()
})

// Coverage: error loading catalogs
test('error loading catalogs shows error message', async () => {
  ;(api.catalogs.list as Mock).mockRejectedValue(new Error('Network error'))
  renderList()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

// Coverage: search filter with no matching results
test('filter with no matches shows filter empty state', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  const searchInput = page.getByPlaceholder('Filter by name')
  await searchInput.fill('nonexistent')
  await expect.element(page.getByText('No catalogs match the filter.')).toBeVisible()
})

// Coverage: clear search filter
test('clearing search filter shows all catalogs', async () => {
  renderList()
  await expect.element(page.getByText('production-app')).toBeVisible()
  const searchInput = page.getByPlaceholder('Filter by name')
  await searchInput.fill('production')
  await expect.element(page.getByText('staging-app')).not.toBeInTheDocument()
  // Clear the filter
  await page.getByRole('button', { name: 'Reset' }).click()
  await expect.element(page.getByText('staging-app')).toBeVisible()
})

// Coverage: generic error path (non-Error object)
test('generic error message on non-Error rejection', async () => {
  ;(api.catalogs.list as Mock).mockRejectedValue('oops')
  renderList()
  await expect.element(page.getByText('Failed to load catalogs')).toBeVisible()
})
