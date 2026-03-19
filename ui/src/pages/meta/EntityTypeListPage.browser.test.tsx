import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter } from 'react-router-dom'
import { EntityTypeListPage } from './EntityTypeListPage'
import { AuthProvider } from '../../context/AuthContext'

const mockEntityTypes = [
  { id: 'et-1', name: 'Model', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'et-2', name: 'Tool', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
  { id: 'et-3', name: 'Prompt', created_at: '2026-01-03T00:00:00Z', updated_at: '2026-01-03T00:00:00Z' },
]

function renderList(role: 'Admin' | 'RO' | 'RW' | 'SuperAdmin' = 'Admin', onNavigate?: (id: string) => void) {
  return render(
    <MemoryRouter>
      <AuthProvider initialRole={role}>
        <EntityTypeListPage entityTypes={mockEntityTypes} total={3} onNavigate={onNavigate} />
      </AuthProvider>
    </MemoryRouter>
  )
}

test('renders all entity types in table', async () => {
  renderList()
  await expect.element(page.getByText('Model')).toBeVisible()
  await expect.element(page.getByText('Tool')).toBeVisible()
  await expect.element(page.getByText('Prompt')).toBeVisible()
  await expect.element(page.getByText('Total: 3')).toBeVisible()
})

test('renders table with data', async () => {
  renderList()
  await expect.element(page.getByText('Model')).toBeVisible()
  await expect.element(page.getByText('Entity Types')).toBeVisible()
})

test('renders filter input', async () => {
  renderList()
  await expect.element(page.getByPlaceholder('Filter by name')).toBeVisible()
})

test('filter narrows displayed entity types', async () => {
  renderList()
  await page.getByPlaceholder('Filter by name').fill('Model')
  await expect.element(page.getByText('Model')).toBeVisible()
  await expect.element(page.getByText('Tool')).not.toBeInTheDocument()
  await expect.element(page.getByText('Prompt')).not.toBeInTheDocument()
})

test('filter is case insensitive', async () => {
  renderList()
  await page.getByPlaceholder('Filter by name').fill('model')
  await expect.element(page.getByText('Model')).toBeVisible()
})

test('clearing filter shows all entity types', async () => {
  renderList()
  await page.getByPlaceholder('Filter by name').fill('Model')
  await expect.element(page.getByText('Tool')).not.toBeInTheDocument()
  await page.getByPlaceholder('Filter by name').fill('')
  await expect.element(page.getByText('Tool')).toBeVisible()
})

test('clicking row calls onNavigate with entity type id', async () => {
  const onNavigate = vi.fn()
  renderList('Admin', onNavigate)
  await page.getByText('Model').click()
  expect(onNavigate).toHaveBeenCalledWith('et-1')
})

test('Admin sees Create and Copy buttons', async () => {
  renderList('Admin')
  await expect.element(page.getByText('Create Entity Type')).toBeVisible()
  await expect.element(page.getByText('Copy Entity Type')).toBeVisible()
})

test('SuperAdmin sees Create and Copy buttons', async () => {
  renderList('SuperAdmin')
  await expect.element(page.getByText('Create Entity Type')).toBeVisible()
  await expect.element(page.getByText('Copy Entity Type')).toBeVisible()
})

test('RO hides Create and Copy buttons', async () => {
  renderList('RO')
  await expect.element(page.getByText('Model')).toBeVisible()
  await expect.element(page.getByText('Create Entity Type')).not.toBeInTheDocument()
  await expect.element(page.getByText('Copy Entity Type')).not.toBeInTheDocument()
})

test('RW hides Create and Copy buttons', async () => {
  renderList('RW')
  await expect.element(page.getByText('Model')).toBeVisible()
  await expect.element(page.getByText('Create Entity Type')).not.toBeInTheDocument()
  await expect.element(page.getByText('Copy Entity Type')).not.toBeInTheDocument()
})

// TD-43: Description column deferred — API does not return entity type description yet

test('renders empty table when no entity types', async () => {
  render(
    <MemoryRouter>
      <AuthProvider initialRole="Admin">
        <EntityTypeListPage entityTypes={[]} total={0} />
      </AuthProvider>
    </MemoryRouter>
  )
  await expect.element(page.getByText('Total: 0')).toBeVisible()
})
