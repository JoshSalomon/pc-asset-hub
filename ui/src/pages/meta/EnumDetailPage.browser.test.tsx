import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import EnumDetailPage from './EnumDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    enums: {
      get: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      listValues: vi.fn(),
      addValue: vi.fn(),
      removeValue: vi.fn(),
      reorderValues: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockEnum = {
  id: 'enum-1',
  name: 'Status',
  description: 'Deployment status values',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

const mockValues = [
  { id: 'v1', value: 'active', ordinal: 0 },
  { id: 'v2', value: 'inactive', ordinal: 1 },
  { id: 'v3', value: 'pending', ordinal: 2 },
]

function renderDetail(role: 'Admin' | 'RO' | 'SuperAdmin' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/enums/enum-1']}>
      <Routes>
        <Route path="/schema/enums/:id" element={<EnumDetailPage role={role} />} />
        <Route path="/schema/enums" element={<div>Enum List</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.enums.get as Mock).mockResolvedValue(mockEnum)
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: mockValues, total: 3 })
  ;(api.enums.update as Mock).mockResolvedValue({ status: 'updated' })
  ;(api.enums.delete as Mock).mockResolvedValue(undefined)
  ;(api.enums.addValue as Mock).mockResolvedValue({ status: 'added' })
  ;(api.enums.removeValue as Mock).mockResolvedValue(undefined)
  ;(api.enums.reorderValues as Mock).mockResolvedValue({ status: 'reordered' })
})

// === T-C.48: Navigate to enum detail ===

test('T-C.48: shows enum name, ID, and dates', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('enum-1')).toBeVisible()
  await expect.element(page.getByText('Name')).toBeVisible()
  await expect.element(page.getByText('ID')).toBeVisible()
})

test('shows back link', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: /Back to Enums/i })).toBeVisible()
})

test('shows values table', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()
  await expect.element(page.getByText('inactive')).toBeVisible()
  await expect.element(page.getByText('pending')).toBeVisible()
})

test('shows empty state when no values', async () => {
  ;(api.enums.listValues as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No values defined yet.')).toBeVisible()
})

test('shows error when load fails', async () => {
  ;(api.enums.get as Mock).mockRejectedValue(new Error('404: not found'))
  renderDetail()
  await expect.element(page.getByText('404: not found')).toBeVisible()
})

// === Admin controls visible ===

test('Admin sees Edit, Delete Enum, Add Value, Remove, and reorder buttons', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit name' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete Enum' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Value' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Remove' }).first()).toBeVisible()
})

test('RO hides edit controls', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await expect.element(page.getByRole('button', { name: 'Edit name' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete Enum' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Add Value' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Remove' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Move up' })).not.toBeInTheDocument()
})

// === Edit name ===

test('edit enum name via modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit name' }).click()
  await expect.element(page.getByText('Edit Enum Name')).toBeVisible()

  await page.getByRole('textbox', { name: /Name/i }).clear()
  await page.getByRole('textbox', { name: /Name/i }).fill('New Status')
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.enums.update).toHaveBeenCalledWith('enum-1', { name: 'New Status', description: 'Deployment status values' })
})

test('rename enum with undefined description sends empty string not undefined', async () => {
  ;(api.enums.get as Mock).mockResolvedValue({ ...mockEnum, description: undefined })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit name' }).click()
  await page.getByRole('textbox', { name: /Name/i }).clear()
  await page.getByRole('textbox', { name: /Name/i }).fill('Renamed')
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  // Must send empty string, not undefined (which JSON.stringify would omit, causing backend to clear description)
  expect(api.enums.update).toHaveBeenCalledWith('enum-1', { name: 'Renamed', description: '' })
})

test('edit name error shown in modal', async () => {
  ;(api.enums.update as Mock).mockRejectedValue(new Error('409: conflict'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit name' }).click()
  await page.getByRole('textbox', { name: /Name/i }).clear()
  await page.getByRole('textbox', { name: /Name/i }).fill('Dup')
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('409: conflict')).toBeVisible()
})

test('edit name cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit name' }).click()
  await expect.element(page.getByText('Edit Enum Name')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// === T-C.49: Add value ===

test('T-C.49: add value to enum', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Add Value' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  await page.getByRole('textbox', { name: /Value/i }).fill('archived')
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  expect(api.enums.addValue).toHaveBeenCalledWith('enum-1', 'archived')
})

test('add value error shown', async () => {
  ;(api.enums.addValue as Mock).mockRejectedValue(new Error('400: bad'))
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Add Value' }).click()
  await page.getByRole('textbox', { name: /Value/i }).fill('bad')
  await page.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  await expect.element(page.getByText('400: bad')).toBeVisible()
})

test('add value cancel closes modal', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Add Value' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// === T-C.50: Remove value ===

test('T-C.50: remove value from enum', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).first().click()
  expect(api.enums.removeValue).toHaveBeenCalledWith('enum-1', 'v1')
})

// === T-C.51: Reorder values ===

test('T-C.51: reorder enum values with down button', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  // Click "Move down" on first value
  await page.getByRole('button', { name: 'Move down' }).first().click()
  expect(api.enums.reorderValues).toHaveBeenCalledWith('enum-1', ['v2', 'v1', 'v3'])
})

test('reorder up button disabled for first item', async () => {
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  const moveUpButtons = page.getByRole('button', { name: 'Move up' })
  await expect.element(moveUpButtons.first()).toBeDisabled()
})

test('reorder down button disabled for last item', async () => {
  renderDetail()
  await expect.element(page.getByText('pending')).toBeVisible()

  const moveDownButtons = page.getByRole('button', { name: 'Move down' })
  // Last "Move down" button (3rd) should be disabled
  await expect.element(moveDownButtons.nth(2)).toBeDisabled()
})

// === T-C.52: Delete referenced enum shows error ===

test('T-C.52: delete enum shows confirmation', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete Enum' }).click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByRole('dialog').getByText('Status')).toBeVisible()
})

test('delete enum confirm calls API', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete Enum' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  expect(api.enums.delete).toHaveBeenCalledWith('enum-1')
})

test('delete enum cancel does nothing', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete Enum' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

  expect(api.enums.delete).not.toHaveBeenCalled()
})

test('delete referenced enum shows error in modal', async () => {
  ;(api.enums.delete as Mock).mockRejectedValue(new Error('422: enum referenced by attributes'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete Enum' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('422: enum referenced by attributes')).toBeVisible()
})

// === SuperAdmin ===

test('SuperAdmin sees all edit controls', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit name' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete Enum' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Add Value' })).toBeVisible()
})

// === Error paths ===

test('remove value error shows alert', async () => {
  ;(api.enums.removeValue as Mock).mockRejectedValue(new Error('500: remove failed'))
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Remove' }).first().click()
  await expect.element(page.getByText('500: remove failed')).toBeVisible()
})

test('reorder value error shows alert', async () => {
  ;(api.enums.reorderValues as Mock).mockRejectedValue(new Error('500: reorder failed'))
  renderDetail()
  await expect.element(page.getByRole('gridcell', { name: 'active', exact: true })).toBeVisible()

  await page.getByRole('button', { name: 'Move down' }).first().click()
  await expect.element(page.getByText('500: reorder failed')).toBeVisible()
})

// Enum description edit via prompt
test('edit enum description via prompt', async () => {
  const origPrompt = window.prompt
  window.prompt = vi.fn().mockReturnValue('Updated desc') as typeof window.prompt
  ;(api.enums.update as Mock).mockResolvedValue({ status: 'updated' })

  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await page.getByRole('button', { name: 'Edit description' }).click()

  expect(window.prompt).toHaveBeenCalled()
  expect(api.enums.update).toHaveBeenCalledWith('enum-1', { name: 'Status', description: 'Updated desc' })

  window.prompt = origPrompt
})

// Fix 5: Enum detail page shows description
test('enum detail shows description', async () => {
  renderDetail()
  await expect.element(page.getByText('Deployment status values')).toBeVisible()
})

// Fix 6: Empty description shows placeholder
test('enum detail shows placeholder for empty description', async () => {
  ;(api.enums.get as Mock).mockResolvedValue({ ...mockEnum, description: '' })
  renderDetail()
  await expect.element(page.getByText('No description')).toBeVisible()
})
