import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import MigrationPreviewModal from './MigrationPreviewModal'
import type { MigrationReport } from '../types'

const baseReport: MigrationReport = {
  affected_catalogs: 2,
  affected_instances: 47,
  attribute_mappings: [
    { old_name: 'endpoint', new_name: 'endpoint', action: 'remap' },
    { old_name: 'old_field', action: 'orphaned' },
  ],
  warnings: [
    { type: 'deleted_attribute', attribute: 'old_field', affected_instances: 12 },
    { type: 'type_changed', attribute: 'port', affected_instances: 47, old_type: 'string', new_type: 'integer' },
    { type: 'new_required', attribute: 'region', affected_instances: 47 },
  ],
}

// T-29.61: Changing pin version shows migration warnings
test('T-29.61: migration preview modal shows warnings and affected counts', async () => {
  const onConfirm = vi.fn()
  const onCancel = vi.fn()
  render(
    <MigrationPreviewModal
      isOpen
      report={baseReport}
      entityTypeName="Server"
      onConfirm={onConfirm}
      onCancel={onCancel}
    />
  )

  const dialog = page.getByRole('dialog')
  await expect.element(dialog).toBeVisible()
  // Header
  await expect.element(dialog.getByText('Migration Preview')).toBeVisible()
  // Entity type name in summary
  await expect.element(dialog.getByText('Server')).toBeVisible()
  // Warnings — check the "Affects N instance(s)" children text
  await expect.element(dialog.getByText('Affects 12 instance(s)')).toBeVisible()
  // Apply button present
  await expect.element(dialog.getByRole('button', { name: 'Apply Change' })).toBeVisible()
})

// T-29.62: Migration warnings show attribute names and affected instance counts
test('T-29.62: migration preview shows attribute mapping table', async () => {
  render(
    <MigrationPreviewModal
      isOpen
      report={baseReport}
      entityTypeName="Server"
      onConfirm={vi.fn()}
      onCancel={vi.fn()}
    />
  )

  // Mapping table
  await expect.element(page.getByRole('gridcell', { name: 'endpoint' }).first()).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'remap' })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'orphaned' })).toBeVisible()
  // Warning details
  await expect.element(page.getByText('Affects 12 instance(s)')).toBeVisible()
  await expect.element(page.getByText('Affects 47 instance(s)').first()).toBeVisible()
})

// T-29.63: Dry-run preview available — confirm applies, cancel aborts
test('T-29.63: confirm calls onConfirm, cancel calls onCancel', async () => {
  const onConfirm = vi.fn()
  const onCancel = vi.fn()
  render(
    <MigrationPreviewModal
      isOpen
      report={baseReport}
      entityTypeName="Server"
      onConfirm={onConfirm}
      onCancel={onCancel}
    />
  )

  await page.getByRole('button', { name: 'Apply Change' }).click()
  expect(onConfirm).toHaveBeenCalledOnce()

  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(onCancel).toHaveBeenCalledOnce()
})

// No warnings, no instances — shows clean message
test('migration preview with no impact shows clean message', async () => {
  const cleanReport: MigrationReport = {
    affected_catalogs: 0,
    affected_instances: 0,
    attribute_mappings: [],
    warnings: [],
  }
  render(
    <MigrationPreviewModal
      isOpen
      report={cleanReport}
      entityTypeName="Server"
      onConfirm={vi.fn()}
      onCancel={vi.fn()}
    />
  )

  await expect.element(page.getByText('No instance data will be affected')).toBeVisible()
})
