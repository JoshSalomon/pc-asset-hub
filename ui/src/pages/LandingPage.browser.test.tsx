import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import LandingPage from './LandingPage'
import { api } from '../api/client'

vi.mock('../api/client', () => ({
  api: {
    catalogs: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockCatalogs = [
  {
    id: 'c1', name: 'prod-catalog', description: 'Production data',
    catalog_version_id: 'cv1', catalog_version_label: 'v2.0',
    validation_status: 'valid', published: true, published_at: '2026-01-01T00:00:00Z',
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'c2', name: 'staging', description: '',
    catalog_version_id: 'cv1', catalog_version_label: 'v2.0',
    validation_status: 'draft',
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'c3', name: 'broken-data', description: 'Bad data',
    catalog_version_id: 'cv1', catalog_version_label: 'v1.0',
    validation_status: 'invalid',
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
]

function renderLanding() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route path="/" element={<LandingPage role="Admin" />} />
        <Route path="/schema" element={<div>Schema Page</div>} />
        <Route path="/catalogs/:name" element={<div>Catalog Viewer</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: mockCatalogs, total: 3 })
})

// T-22.07: Landing page renders
test('T-22.07: landing page renders', async () => {
  renderLanding()
  await expect.element(page.getByText('Schema Management')).toBeVisible()
})

// T-22.08: Schema Management section is visible with card
test('T-22.08: Schema Management section is visible', async () => {
  renderLanding()
  await expect.element(page.getByRole('heading', { name: /Schema Management/i })).toBeVisible()
  await expect.element(page.getByText('Entity Types & Model')).toBeVisible()
})

// T-22.10: Catalog cards rendered for each catalog
test('T-22.10: catalog cards rendered for each catalog', async () => {
  renderLanding()
  await expect.element(page.getByText('prod-catalog')).toBeVisible()
  await expect.element(page.getByText('staging')).toBeVisible()
  await expect.element(page.getByText('broken-data')).toBeVisible()
})

// T-22.11: Catalog card shows name, CV label, status badge
test('T-22.11: catalog card shows name, CV label, status', async () => {
  renderLanding()
  await expect.element(page.getByText('prod-catalog')).toBeVisible()
  await expect.element(page.getByText('v2.0').first()).toBeVisible()
  await expect.element(page.getByText('valid', { exact: true }).first()).toBeVisible()
})

// T-22.05: Published catalog shows published indicator
test('T-22.05: published catalog shows published indicator', async () => {
  renderLanding()
  await expect.element(page.getByText('Published').first()).toBeVisible()
})

// T-22.02: Draft status badge renders blue
test('T-22.02: draft badge visible', async () => {
  renderLanding()
  await expect.element(page.getByText('draft', { exact: true })).toBeVisible()
})

// T-22.04: Invalid status badge renders red
test('T-22.04: invalid badge visible', async () => {
  renderLanding()
  await expect.element(page.getByText('invalid', { exact: true })).toBeVisible()
})

// T-22.06: Card with no description renders cleanly
test('T-22.06: card with no description renders cleanly', async () => {
  renderLanding()
  // 'staging' has empty description — should render without crash
  await expect.element(page.getByText('staging')).toBeVisible()
})

// T-22.09: Schema Management card links to /schema
test('T-22.09: clicking Schema Management card navigates to /schema', async () => {
  renderLanding()
  await page.getByText('Entity Types & Model').click()
  await expect.element(page.getByText('Schema Page')).toBeVisible()
})

// T-22.12: Clicking catalog card navigates to /catalogs/:name
test('T-22.12: clicking catalog card navigates to catalog viewer', async () => {
  renderLanding()
  await expect.element(page.getByText('prod-catalog')).toBeVisible()
  await page.getByText('prod-catalog').click()
  await expect.element(page.getByText('Catalog Viewer')).toBeVisible()
})

// T-22.13: Empty state when no catalogs
test('T-22.13: empty state when no catalogs', async () => {
  ;(api.catalogs.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderLanding()
  await expect.element(page.getByText(/no catalogs/i)).toBeVisible()
})

// T-22.15: Error state on API failure
test('T-22.15: error state on API failure', async () => {
  ;(api.catalogs.list as Mock).mockRejectedValue(new Error('Network error'))
  renderLanding()
  await expect.element(page.getByText('Network error')).toBeVisible()
})
