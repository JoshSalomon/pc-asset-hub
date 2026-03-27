import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EnumListPage from './EnumListPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    enums: {
      list: vi.fn(),
      create: vi.fn(),
      delete: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockEnums = [
  { id: 'enum-1', name: 'Status', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'enum-2', name: 'Priority', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
]

function renderList(role: 'Admin' | 'RO' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/enums']}>
      <Routes>
        <Route path="/schema/enums" element={<EnumListPage role={role} />} />
        <Route path="/schema/enums/:id" element={<div>Enum Detail</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.enums.list as Mock).mockResolvedValue({ items: mockEnums, total: 2 })
  ;(api.enums.create as Mock).mockResolvedValue({ id: 'enum-3', name: 'NewEnum', created_at: '2026-01-03T00:00:00Z', updated_at: '2026-01-03T00:00:00Z' })
  ;(api.enums.delete as Mock).mockResolvedValue(undefined)
})

// === T-C.46: Enum list page shows enums ===

test('T-C.46: shows enum list', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()
  await expect.element(page.getByText('Priority')).toBeVisible()
})

test('shows empty state when no enums', async () => {
  ;(api.enums.list as Mock).mockResolvedValue({ items: [], total: 0 })
  renderList()
  await expect.element(page.getByText('No enums yet. Create one to get started.')).toBeVisible()
})

test('shows error when list fails', async () => {
  ;(api.enums.list as Mock).mockRejectedValue(new Error('500: server error'))
  renderList()
  await expect.element(page.getByText('500: server error')).toBeVisible()
})

test('enum names are clickable links', async () => {
  renderList()
  await expect.element(page.getByRole('button', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Priority' })).toBeVisible()
})

test('clicking enum name navigates to detail', async () => {
  renderList()
  await expect.element(page.getByRole('button', { name: 'Status' })).toBeVisible()
  await page.getByRole('button', { name: 'Status' }).click()
  await expect.element(page.getByText('Enum Detail')).toBeVisible()
})

// === T-C.47: Create enum with initial values ===

test('T-C.47: create enum with initial values', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Create Enum' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('textbox', { name: /Name/i }).fill('NewEnum')
  await page.getByPlaceholder('Optional description').fill('Test enum desc')
  await page.getByPlaceholder('e.g. active, inactive, pending').fill('a, b, c')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.enums.create).toHaveBeenCalledWith({ name: 'NewEnum', description: 'Test enum desc', values: ['a', 'b', 'c'] })
})

test('create enum without values', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Create Enum' }).click()
  await page.getByRole('textbox', { name: /Name/i }).fill('EmptyEnum')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  expect(api.enums.create).toHaveBeenCalledWith({ name: 'EmptyEnum', values: undefined })
})

test('create enum error shown in modal', async () => {
  ;(api.enums.create as Mock).mockRejectedValue(new Error('409: name exists'))
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Create Enum' }).click()
  await page.getByRole('textbox', { name: /Name/i }).fill('Status')
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await expect.element(page.getByText('409: name exists')).toBeVisible()
})

test('create enum cancel closes modal', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Create Enum' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// === Delete confirmation ===

test('delete enum with confirmation', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Status')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.enums.delete).toHaveBeenCalledWith('enum-1')
})

test('delete cancel does nothing', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

  expect(api.enums.delete).not.toHaveBeenCalled()
})

test('delete error shown in modal', async () => {
  ;(api.enums.delete as Mock).mockRejectedValue(new Error('422: enum referenced'))
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('422: enum referenced')).toBeVisible()
})

// === Refresh ===

test('refresh reloads enums', async () => {
  renderList()
  await expect.element(page.getByText('Status')).toBeVisible()

  const callsBefore = (api.enums.list as Mock).mock.calls.length
  await page.getByRole('button', { name: 'Refresh' }).click()

  await vi.waitFor(() => {
    expect((api.enums.list as Mock).mock.calls.length).toBeGreaterThan(callsBefore)
  })
})

// === RBAC ===

test('RO hides Create and Delete buttons', async () => {
  renderList('RO')
  await expect.element(page.getByText('Status')).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Create Enum' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})
