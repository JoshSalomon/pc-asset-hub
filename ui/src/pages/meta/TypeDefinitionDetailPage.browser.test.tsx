import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import TypeDefinitionDetailPage from './TypeDefinitionDetailPage'
import { api } from '../../api/client'

vi.mock('../../api/client', () => ({
  api: {
    typeDefinitions: {
      get: vi.fn(),
      listVersions: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockTypeDef = {
  id: 'td-1',
  name: 'Status',
  description: 'Allowed status values',
  base_type: 'enum' as const,
  system: false,
  latest_version: 2, latest_version_id: 'tdv-auto',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

const mockSystemTypeDef = {
  ...mockTypeDef,
  id: 'td-sys',
  name: 'SystemType',
  system: true,
}

const mockVersions = [
  {
    id: 'tdv-1',
    type_definition_id: 'td-1',
    version_number: 1,
    constraints: { allowed_values: ['active', 'inactive'] },
    created_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'tdv-2',
    type_definition_id: 'td-1',
    version_number: 2,
    constraints: { allowed_values: ['active', 'inactive', 'archived'] },
    created_at: '2026-01-02T00:00:00Z',
  },
]

const mockVersionsNoConstraints = [
  {
    id: 'tdv-1',
    type_definition_id: 'td-1',
    version_number: 1,
    constraints: {},
    created_at: '2026-01-01T00:00:00Z',
  },
]

function renderDetail(role: 'Admin' | 'RO' | 'RW' | 'SuperAdmin' = 'Admin') {
  return render(
    <MemoryRouter initialEntries={['/schema/types/td-1']}>
      <Routes>
        <Route path="/schema/types/:id" element={<TypeDefinitionDetailPage role={role} />} />
        <Route path="/schema/types" element={<div>Types List Page</div>} />
      </Routes>
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.typeDefinitions.get as Mock).mockResolvedValue(mockTypeDef)
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: mockVersions, total: 2 })
  ;(api.typeDefinitions.update as Mock).mockResolvedValue({
    id: 'tdv-3',
    type_definition_id: 'td-1',
    version_number: 3,
    constraints: {},
    created_at: '2026-01-03T00:00:00Z',
  })
  ;(api.typeDefinitions.delete as Mock).mockResolvedValue(undefined)
})

// === Unreachable guards (if (!id) return) ===
//
// Lines 61, 83, 95, 116: All are `if (!id) return` guards in loadTypeDef,
// handleUpdateDescription, handleUpdateConstraints, and handleDelete respectively.
// The `id` comes from useParams<{ id: string }>() which always has a value when
// the route pattern `/schema/types/:id` matches. Since this page component is only
// rendered when that route matches, `id` is always defined. These guards are
// defensive code that cannot be triggered through normal routing.

// === Rendering basic details ===

test('renders type definition details (name, description, base type)', async () => {
  // Use only one version to avoid "V2" appearing in multiple places
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({
    items: [mockVersions[0]],
    total: 1,
  })
  ;(api.typeDefinitions.get as Mock).mockResolvedValue({ ...mockTypeDef, latest_version: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('Allowed status values')).toBeVisible()
  await expect.element(page.getByText('enum')).toBeVisible()
  // Latest version shown in description list
  await expect.element(page.getByText('V1', { exact: true }).first()).toBeVisible()
  await expect.element(page.getByText('td-1')).toBeVisible()
})

test('renders "No description" when description is empty', async () => {
  ;(api.typeDefinitions.get as Mock).mockResolvedValue({ ...mockTypeDef, description: '' })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No description')).toBeVisible()
})

test('renders "No description" when description is undefined', async () => {
  const tdNoDesc = { ...mockTypeDef }
  delete (tdNoDesc as Record<string, unknown>).description
  ;(api.typeDefinitions.get as Mock).mockResolvedValue(tdNoDesc)
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No description')).toBeVisible()
})

// === System badge ===

test('shows "System" badge for system types', async () => {
  ;(api.typeDefinitions.get as Mock).mockResolvedValue(mockSystemTypeDef)
  renderDetail()
  await expect.element(page.getByRole('heading', { name: /SystemType/ })).toBeVisible()
  await expect.element(page.getByText('System', { exact: true })).toBeVisible()
})

test('does not show "System" badge for non-system types', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('System')).not.toBeInTheDocument()
})

// === System type read-only behavior ===

test('system types: edit buttons are hidden', async () => {
  ;(api.typeDefinitions.get as Mock).mockResolvedValue(mockSystemTypeDef)
  renderDetail()
  await expect.element(page.getByRole('heading', { name: /SystemType/ })).toBeVisible()
  // Edit description button should not be present
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
  // Edit Constraints button should not be present
  await expect.element(page.getByRole('button', { name: 'Edit Constraints' })).not.toBeInTheDocument()
  // Delete button should not be present
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

// === Role-based controls ===

test('non-system type with Admin role shows edit controls', async () => {
  renderDetail('Admin')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit Constraints' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('non-system type with SuperAdmin role shows edit controls', async () => {
  renderDetail('SuperAdmin')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit Constraints' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Delete' })).toBeVisible()
})

test('RO role hides edit controls', async () => {
  renderDetail('RO')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Edit Constraints' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

test('RW role hides edit controls', async () => {
  renderDetail('RW')
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Edit description' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Edit Constraints' })).not.toBeInTheDocument()
  await expect.element(page.getByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
})

// === Loading and error states ===

test('shows loading spinner while fetching', async () => {
  ;(api.typeDefinitions.get as Mock).mockReturnValue(new Promise(() => {})) // never resolves
  ;(api.typeDefinitions.listVersions as Mock).mockReturnValue(new Promise(() => {}))
  renderDetail()
  await expect.element(page.getByLabelText('Loading')).toBeVisible()
})

test('error state shows alert when API fails', async () => {
  ;(api.typeDefinitions.get as Mock).mockRejectedValue(new Error('Network error'))
  ;(api.typeDefinitions.listVersions as Mock).mockRejectedValue(new Error('Network error'))
  renderDetail()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

test('error state with non-Error object', async () => {
  ;(api.typeDefinitions.get as Mock).mockRejectedValue('something went wrong')
  ;(api.typeDefinitions.listVersions as Mock).mockRejectedValue('something went wrong')
  renderDetail()
  await expect.element(page.getByText('Failed to load')).toBeVisible()
})

test('type not found shows warning alert', async () => {
  ;(api.typeDefinitions.get as Mock).mockResolvedValue(null)
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByText('Type definition not found')).toBeVisible()
})

// === Navigate back ===

test('back button navigates to types list', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await page.getByRole('button', { name: /Back to Types/ }).click()
  await expect.element(page.getByText('Types List Page')).toBeVisible()
})

// === Edit description ===

test('edit description: opens inline editor, saves, and reloads', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  // Click Edit button to open inline description editor
  await page.getByRole('button', { name: 'Edit description' }).click()

  // The text input should appear with the current description
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await expect.element(descInput).toBeVisible()

  // Clear and type new value
  await descInput.fill('Updated status description')

  // Click Save
  await page.getByRole('button', { name: 'Save' }).click()

  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', { description: 'Updated status description' })
})

test('edit description: cancel closes inline editor without saving', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit description' }).click()
  await expect.element(page.getByRole('textbox', { name: 'Description' })).toBeVisible()

  await page.getByRole('button', { name: 'Cancel' }).click()

  // Editor should be closed
  await expect.element(page.getByRole('textbox', { name: 'Description' })).not.toBeInTheDocument()
  expect(api.typeDefinitions.update).not.toHaveBeenCalled()
})

test('edit description: shows error on failure', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue(new Error('Update failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit description' }).click()
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await descInput.fill('new desc')
  await page.getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('Update failed')).toBeVisible()
})

test('edit description: shows fallback error on non-Error rejection', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue('bad')
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit description' }).click()
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await descInput.fill('new desc')
  await page.getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('Failed to update')).toBeVisible()
})

// Cancel clears the description error
test('edit description: cancel clears error state', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue(new Error('Update failed'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit description' }).click()
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await descInput.fill('new desc')
  await page.getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('Update failed')).toBeVisible()

  // Cancel should clear the error
  await page.getByRole('button', { name: 'Cancel' }).click()

  // Re-open to verify error was cleared
  await page.getByRole('button', { name: 'Edit description' }).click()
  await expect.element(page.getByText('Update failed')).not.toBeInTheDocument()
})

// === Edit constraints ===

test('edit constraints: opens modal, saves, and reloads', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()

  // Modal should open
  await expect.element(page.getByText('Edit Constraints (creates new version)')).toBeVisible()

  // Click Save in the modal
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', {
    constraints: expect.any(Object),
  })
})

test('edit constraints: cancel closes modal without saving', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await expect.element(page.getByText('Edit Constraints (creates new version)')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
  expect(api.typeDefinitions.update).not.toHaveBeenCalled()
})

test('edit constraints: shows error on failure', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue(new Error('Constraints error'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('Constraints error')).toBeVisible()
})

test('edit constraints: shows fallback error on non-Error rejection', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue('bad')
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  await expect.element(page.getByText('Failed to update constraints')).toBeVisible()
})

test('edit constraints: close modal via onClose clears error', async () => {
  ;(api.typeDefinitions.update as Mock).mockRejectedValue(new Error('Constraints error'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
  await expect.element(page.getByText('Constraints error')).toBeVisible()

  // Close via X button
  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

test('edit constraints: cleans up empty constraint values', async () => {
  // Use a type with no current constraints so we save an empty object
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: mockVersionsNoConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', {
    constraints: {},
  })
})

// === Version history ===

test('version history table shows all versions', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('Version History')).toBeVisible()

  // Check that the table rendered with aria-label
  await expect.element(page.getByLabelText('Type definition versions')).toBeVisible()

  // Both version rows: check constraint JSON is visible in cells
  const v1Constraints = JSON.stringify({ allowed_values: ['active', 'inactive'] })
  await expect.element(page.getByText(v1Constraints)).toBeVisible()
  const v2Constraints = JSON.stringify({ allowed_values: ['active', 'inactive', 'archived'] })
  await expect.element(page.getByText(v2Constraints)).toBeVisible()
})

test('version history shows constraints for each version', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  // Version 1 constraints
  const v1Constraints = JSON.stringify({ allowed_values: ['active', 'inactive'] })
  await expect.element(page.getByText(v1Constraints)).toBeVisible()

  // Version 2 constraints
  const v2Constraints = JSON.stringify({ allowed_values: ['active', 'inactive', 'archived'] })
  await expect.element(page.getByText(v2Constraints)).toBeVisible()
})

test('version history shows "None" for versions without constraints', async () => {
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: mockVersionsNoConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('None')).toBeVisible()
})

test('version history shows empty state when no versions', async () => {
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No versions found.')).toBeVisible()
})

// === Current constraints display ===

test('shows current constraints section when latest version has constraints', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByRole('heading', { name: /Current Constraints/ })).toBeVisible()
  // The constraint key appears in the description list term
  // Use .first() to avoid matching JSON in version table cells
  await expect.element(page.getByText('allowed_values', { exact: true }).first()).toBeVisible()
  await expect.element(page.getByText('active, inactive, archived')).toBeVisible()
})

test('hides current constraints section when latest version has no constraints', async () => {
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: mockVersionsNoConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText(/Current Constraints/)).not.toBeInTheDocument()
})

test('shows non-array constraint values as strings', async () => {
  const versionsWithScalarConstraints = [
    {
      id: 'tdv-1',
      type_definition_id: 'td-1',
      version_number: 1,
      constraints: { max_length: 255, pattern: '^[a-z]+$' },
      created_at: '2026-01-01T00:00:00Z',
    },
  ]
  ;(api.typeDefinitions.get as Mock).mockResolvedValue({ ...mockTypeDef, base_type: 'string', latest_version: 1 })
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: versionsWithScalarConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  // Use .first() since "255" appears in both the Current Constraints section and the table cell JSON
  await expect.element(page.getByText('255', { exact: true }).first()).toBeVisible()
  await expect.element(page.getByText('^[a-z]+$', { exact: true }).first()).toBeVisible()
})

// === Delete ===

test('delete confirmation: confirm deletes and navigates to list', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()
  await expect.element(page.getByText(/Are you sure you want to delete type definition/)).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  expect(api.typeDefinitions.delete).toHaveBeenCalledWith('td-1')

  // Should navigate to the types list
  await expect.element(page.getByText('Types List Page')).toBeVisible()
})

test('delete confirmation: cancel does not delete', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Confirm Deletion')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
  expect(api.typeDefinitions.delete).not.toHaveBeenCalled()
})

test('delete failure shows error in modal', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue(new Error('Cannot delete'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('Cannot delete')).toBeVisible()
})

test('delete failure with non-Error shows fallback message', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue('bad')
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await expect.element(page.getByText('Failed to delete')).toBeVisible()
})

test('delete modal close via X button clears error', async () => {
  ;(api.typeDefinitions.delete as Mock).mockRejectedValue(new Error('Cannot delete'))
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Delete' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await expect.element(page.getByText('Cannot delete')).toBeVisible()

  await page.getByRole('dialog').getByRole('button', { name: 'Close' }).click()
  await expect.element(page.getByRole('dialog')).not.toBeInTheDocument()
})

// === Error displayed with loaded typeDef (error && typeDef case) ===

test('shows inline error alert when reload fails after initial load', async () => {
  let callCount = 0
  ;(api.typeDefinitions.get as Mock).mockImplementation(() => {
    callCount++
    if (callCount === 1) return Promise.resolve(mockTypeDef)
    return Promise.reject(new Error('Reload failed'))
  })
  ;(api.typeDefinitions.listVersions as Mock).mockImplementation(() => {
    if (callCount === 1) return Promise.resolve({ items: mockVersions, total: 2 })
    return Promise.reject(new Error('Reload failed'))
  })

  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  // Trigger reload by editing description and saving
  await page.getByRole('button', { name: 'Edit description' }).click()
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await descInput.fill('new desc')
  await page.getByRole('button', { name: 'Save' }).click()

  // After the update succeeds, loadTypeDef is called again which now fails
  // The inline error should be visible along with the stale typeDef data
  await expect.element(page.getByText('Reload failed')).toBeVisible()
  // Type def heading should still be visible (stale data)
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
})

// === Version items null safety ===

test('handles null items from listVersions gracefully', async () => {
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: null, total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No versions found.')).toBeVisible()
})

// === Edit constraints with non-null latest version having constraints ===

test('edit constraints pre-fills with current constraints from latest version', async () => {
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()

  // The modal should be open and we can save whatever was pre-filled
  await expect.element(page.getByText('Edit Constraints (creates new version)')).toBeVisible()

  // Save pre-filled constraints
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  // Should have cleaned constraints (allowed_values is an array which is truthy, so kept)
  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', {
    constraints: { allowed_values: ['active', 'inactive', 'archived'] },
  })
})

test('edit constraints with no versions opens with empty constraints', async () => {
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', {
    constraints: {},
  })
})

// === Created date formatting ===

test('renders formatted created date', async () => {
  // Use no versions to avoid date appearing in both description list and table
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: [], total: 0 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  // The created_at is '2026-01-01T00:00:00Z' - just check the date appears in some format
  const createdDate = new Date('2026-01-01T00:00:00Z').toLocaleString()
  await expect.element(page.getByText(createdDate)).toBeVisible()
})

// === Additional branch coverage ===

test('edit description button works when typeDef has no description (empty fallback)', async () => {
  // This covers the branch on line 177: setEditDescValue(typeDef.description || '')
  ;(api.typeDefinitions.get as Mock).mockResolvedValue({ ...mockTypeDef, description: undefined })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  await expect.element(page.getByText('No description')).toBeVisible()

  // Click Edit button - should open with empty string since description is undefined
  await page.getByRole('button', { name: 'Edit description' }).click()
  const descInput = page.getByRole('textbox', { name: 'Description' })
  await expect.element(descInput).toBeVisible()

  // Save with the default empty value
  await page.getByRole('button', { name: 'Save' }).click()
  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', { description: '' })
})

test('edit constraints cleans up null, undefined, and empty string values', async () => {
  // Covers the branch on line 101: filtering out empty/null/undefined constraint values
  // Use string type so the constraints form has inputs we can manipulate
  const versionsWithMixedConstraints = [
    {
      id: 'tdv-1',
      type_definition_id: 'td-1',
      version_number: 1,
      constraints: { max_length: 10, pattern: '', multiline: null as unknown, extra: undefined as unknown },
      created_at: '2026-01-01T00:00:00Z',
    },
  ]
  ;(api.typeDefinitions.get as Mock).mockResolvedValue({ ...mockTypeDef, base_type: 'string', latest_version: 1 })
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: versionsWithMixedConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  await page.getByRole('button', { name: 'Edit Constraints' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

  // Should have cleaned out the empty/null/undefined values, keeping only max_length
  expect(api.typeDefinitions.update).toHaveBeenCalledWith('td-1', {
    constraints: { max_length: 10 },
  })
})

test('latestVersion reduce picks the higher version number when versions are out of order', async () => {
  // Covers the reduce branch on line 134: a.version_number > b.version_number ? a : b
  // Provide versions with higher number first to exercise the a > b branch
  const outOfOrderVersions = [
    {
      id: 'tdv-2',
      type_definition_id: 'td-1',
      version_number: 2,
      constraints: { allowed_values: ['yes', 'no'] },
      created_at: '2026-01-02T00:00:00Z',
    },
    {
      id: 'tdv-1',
      type_definition_id: 'td-1',
      version_number: 1,
      constraints: { allowed_values: ['on', 'off'] },
      created_at: '2026-01-01T00:00:00Z',
    },
  ]
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: outOfOrderVersions, total: 2 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()

  // The "Current Constraints" section should use V2 (the higher version), not V1
  await expect.element(page.getByRole('heading', { name: /Current Constraints \(V2\)/ })).toBeVisible()
  await expect.element(page.getByText('yes, no')).toBeVisible()
})

test('version history renders version with undefined constraints', async () => {
  // Covers the branch on line 252: v.constraints || {} when constraints is undefined
  const versionsWithUndefinedConstraints = [
    {
      id: 'tdv-1',
      type_definition_id: 'td-1',
      version_number: 1,
      constraints: undefined as unknown as Record<string, unknown>,
      created_at: '2026-01-01T00:00:00Z',
    },
  ]
  ;(api.typeDefinitions.listVersions as Mock).mockResolvedValue({ items: versionsWithUndefinedConstraints, total: 1 })
  renderDetail()
  await expect.element(page.getByRole('heading', { name: 'Status' })).toBeVisible()
  // Should show "None" since constraints is undefined
  await expect.element(page.getByText('None')).toBeVisible()
})
