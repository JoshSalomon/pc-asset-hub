import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import ImportCatalogModal from './ImportCatalogModal'
import { api } from '../api/client'

vi.mock('../api/client', () => ({
  api: {
    catalogs: {
      import: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

function renderModal(overrides: Partial<React.ComponentProps<typeof ImportCatalogModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSuccess: vi.fn(),
    ...overrides,
  }
  return { ...render(<ImportCatalogModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// Helper to simulate file upload via the file input
function createJsonFile(content: object, name = 'export.json'): File {
  const blob = new Blob([JSON.stringify(content)], { type: 'application/json' })
  return new File([blob], name, { type: 'application/json' })
}

// Helper to upload a file into the modal via drag-and-drop on the dropzone.
// The native file input onChange doesn't trigger React synthetic events reliably
// in Playwright browser mode, but native drag-and-drop events work because
// the component registers addEventListener directly on the DOM element.
async function uploadFile(fileContent: object, fileName = 'export.json') {
  const file = createJsonFile(fileContent, fileName)
  await new Promise(resolve => setTimeout(resolve, 50))
  const dropzone = document.querySelector('[data-testid="file-dropzone"]') as HTMLElement
  if (!dropzone) throw new Error('Dropzone not found — modal may not be on upload step')
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(file)
  dropzone.dispatchEvent(new DragEvent('dragover', { bubbles: true, dataTransfer }))
  dropzone.dispatchEvent(new DragEvent('drop', { bubbles: true, dataTransfer }))
  // Wait for FileReader to process
  await new Promise(resolve => setTimeout(resolve, 300))
}

const sampleExportData = {
  format_version: '1.0',
  catalog: { name: 'my-catalog' },
  catalog_version: { label: 'v1.0' },
  entity_types: [{ name: 'Server' }],
  type_definitions: [{ name: 'Status' }],
}

const dryRunNoCollisions = {
  status: 'ok',
  collisions: [],
  summary: { total_entities: 3, conflicts: 0, identical: 0, new: 3 },
}

const dryRunWithCollisions = {
  status: 'ok',
  collisions: [
    { type: 'entity_type', name: 'Server', resolution: 'identical', version: 1, detail: 'Same schema' },
    { type: 'type_definition', name: 'Status', resolution: 'conflict', detail: 'Different constraints' },
  ],
  summary: { total_entities: 3, conflicts: 1, identical: 1, new: 1 },
}

const dryRunIdenticalOnly = {
  status: 'ok',
  collisions: [
    { type: 'entity_type', name: 'Server', resolution: 'identical', version: 1, detail: 'Same schema' },
  ],
  summary: { total_entities: 2, conflicts: 0, identical: 1, new: 1 },
}

const importResult = {
  status: 'ok',
  catalog_name: 'my-catalog',
  catalog_id: 'cat-123',
  types_created: 2,
  types_reused: 1,
  instances_created: 5,
  links_created: 3,
}

// === Upload Step ===

test('renders upload step with file input', async () => {
  renderModal()
  await expect.element(page.getByText('Import Catalog')).toBeVisible()
  await expect.element(page.getByText('Catalog File (JSON)')).toBeVisible()
  // Analyze button should be present but disabled (no file selected)
  await expect.element(page.getByRole('button', { name: 'Analyze' })).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Analyze' })).toHaveAttribute('disabled')
})

test('cancel button closes modal', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

test('not rendered when closed', async () => {
  renderModal({ isOpen: false })
  expect(page.getByText('Import Catalog').query()).toBeNull()
})

// === Drag and Drop ===

test('drag and drop file populates catalog name and version fields', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))

  const dropzone = document.querySelector('[data-testid="file-dropzone"]') as HTMLElement
  expect(dropzone).not.toBeNull()

  const file = createJsonFile(sampleExportData, 'dragged.json')
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(file)

  dropzone.dispatchEvent(new DragEvent('dragover', { bubbles: true, dataTransfer }))
  dropzone.dispatchEvent(new DragEvent('drop', { bubbles: true, dataTransfer }))

  await new Promise(resolve => setTimeout(resolve, 300))

  const nameInput = page.getByRole('textbox', { name: /Catalog Name/i })
  await expect.element(nameInput).toBeVisible()
  await expect.element(nameInput).toHaveValue('my-catalog')
  await expect.element(page.getByText('dragged.json')).toBeVisible()
})

test('dropzone shows visual feedback on dragover', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))

  const dropzone = document.querySelector('[data-testid="file-dropzone"]') as HTMLElement
  expect(dropzone).not.toBeNull()

  dropzone.dispatchEvent(new DragEvent('dragenter', { bubbles: true }))
  // Should have a visual indicator (e.g., border change)
  await expect.element(page.getByText(/drop.*here|drag.*file/i)).toBeVisible()
})

test('modal prevents default on document dragover so Chromium allows drops', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))

  const event = new DragEvent('dragover', { bubbles: true, cancelable: true })
  document.dispatchEvent(event)
  expect(event.defaultPrevented).toBe(true)
})

test('drag entering via document body still allows drop on dropzone', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))

  // Simulate Chromium behavior: drag enters through document body first
  document.body.dispatchEvent(new DragEvent('dragenter', { bubbles: true, cancelable: true }))

  const dropzone = document.querySelector('[data-testid="file-dropzone"]') as HTMLElement
  const file = createJsonFile(sampleExportData, 'chrome-drop.json')
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(file)

  dropzone.dispatchEvent(new DragEvent('dragover', { bubbles: true, dataTransfer }))
  dropzone.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer }))

  await new Promise(resolve => setTimeout(resolve, 200))
  await expect.element(page.getByText('chrome-drop.json')).toBeVisible()
})

// === File Upload ===

test('uploading valid JSON shows catalog name and version fields', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  // Should auto-populate catalog name from file
  const nameInput = page.getByRole('textbox', { name: /Catalog Name/i })
  await expect.element(nameInput).toBeVisible()
  await expect.element(nameInput).toHaveValue('my-catalog')
  // CV label should be populated
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await expect.element(cvInput).toHaveValue('v1.0')
})

test('uploading invalid JSON shows error', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))
  const input = document.querySelector('input[type="file"]') as HTMLInputElement
  const blob = new Blob(['not json'], { type: 'application/json' })
  const file = new File([blob], 'bad.json', { type: 'application/json' })
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(file)
  Object.defineProperty(input, 'files', { value: dataTransfer.files, writable: true, configurable: true })
  input.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise(resolve => setTimeout(resolve, 200))

  await expect.element(page.getByText('Invalid JSON file')).toBeVisible()
})

test('uploading file over 50MB shows size error', async () => {
  renderModal()
  await new Promise(resolve => setTimeout(resolve, 50))
  const input = document.querySelector('input[type="file"]') as HTMLInputElement
  // Create a fake file object with size > 50MB
  const bigFile = new File(['x'], 'huge.json', { type: 'application/json' })
  Object.defineProperty(bigFile, 'size', { value: 51 * 1024 * 1024 })
  const dataTransfer = new DataTransfer()
  dataTransfer.items.add(bigFile)
  Object.defineProperty(input, 'files', { value: dataTransfer.files, writable: true, configurable: true })
  input.dispatchEvent(new Event('change', { bubbles: true }))
  await new Promise(resolve => setTimeout(resolve, 200))

  await expect.element(page.getByText(/file too large/i)).toBeVisible()
})

test('uploading non-catalog JSON shows structure error', async () => {
  renderModal()
  await uploadFile({ name: 'my-package', version: '1.0.0', dependencies: {} }, 'package.json')
  await expect.element(page.getByText(/not a valid catalog export/i)).toBeVisible()
})

test('uploading JSON missing catalog_version shows structure error', async () => {
  renderModal()
  await uploadFile({ format_version: '1.0', catalog: { name: 'test' } }, 'partial.json')
  await expect.element(page.getByText(/not a valid catalog export/i)).toBeVisible()
})

test('shows file name after upload', async () => {
  renderModal()
  await uploadFile(sampleExportData, 'my-export.json')
  await expect.element(page.getByText('my-export.json')).toBeVisible()
})

// === Catalog Name Validation ===

test('empty name shows validation error', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const nameInput = page.getByRole('textbox', { name: /Catalog Name/i })
  await expect.element(nameInput).toBeVisible()
  await nameInput.fill('')
  await expect.element(page.getByText('Name is required')).toBeVisible()
})

test('name too long shows validation error', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const nameInput = page.getByRole('textbox', { name: /Catalog Name/i })
  await expect.element(nameInput).toBeVisible()
  await nameInput.fill('a'.repeat(64))
  await expect.element(page.getByText('Name must be at most 63 characters')).toBeVisible()
})

test('invalid DNS label shows validation error', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const nameInput = page.getByRole('textbox', { name: /Catalog Name/i })
  await expect.element(nameInput).toBeVisible()
  await nameInput.fill('INVALID!')
  await expect.element(page.getByText('Must be lowercase alphanumeric and hyphens')).toBeVisible()
})

// === CV Label Validation ===

test('empty CV label shows validation error', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await cvInput.fill('')
  await expect.element(page.getByText('Version label is required')).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Analyze' })).toHaveAttribute('disabled')
})

test('CV label with no alphanumeric characters shows validation error', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await cvInput.fill('***')
  await expect.element(page.getByText(/must contain at least one alphanumeric/i)).toBeVisible()
})

test('CV label with alphanumeric characters is accepted', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await cvInput.fill('v2.0-beta')
  const errorMsg = page.getByText(/must contain at least one alphanumeric/i)
  await expect.element(errorMsg).not.toBeInTheDocument()
})

test('Analyze button disabled when CV label has no alphanumeric characters', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await cvInput.fill('()')
  await expect.element(page.getByRole('button', { name: 'Analyze' })).toHaveAttribute('disabled')
})

// === Analyze (Dry Run) ===

test('analyze calls import API with dry_run and shows confirm on no collisions', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunNoCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  expect(api.catalogs.import).toHaveBeenCalledWith(
    expect.objectContaining({
      catalog_name: 'my-catalog',
      catalog_version_label: 'v1.0',
      data: sampleExportData,
    }),
    { dry_run: true },
  )
  // Should skip to confirm step (no collisions)
  await expect.element(page.getByText(/Ready to import/)).toBeVisible()
})

test('analyze with collisions shows collision table', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunWithCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Collision step should show
  await expect.element(page.getByText(/3 entities/)).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'Server', exact: true })).toBeVisible()
  await expect.element(page.getByRole('gridcell', { name: 'Status', exact: true })).toBeVisible()
  // Identical entity shows checked checkbox, conflict shows unchecked checkbox
  await expect.element(page.getByRole('checkbox', { name: 'Reuse existing' })).toBeChecked()
  await expect.element(page.getByRole('checkbox', { name: 'Create new' })).not.toBeChecked()
})

test('analyze error shows error alert', async () => {
  ;(api.catalogs.import as Mock).mockRejectedValue(new Error('409: conflict'))
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await expect.element(page.getByText('409: conflict')).toBeVisible()
})

test('analyze with non-Error exception shows generic message', async () => {
  ;(api.catalogs.import as Mock).mockRejectedValue('some string error')
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await expect.element(page.getByText('Dry run failed')).toBeVisible()
})

test('analyze skips when no file data', async () => {
  renderModal()
  // Analyze button is disabled, but test the handler guard
  await expect.element(page.getByRole('button', { name: 'Analyze' })).toHaveAttribute('disabled')
})

// === Mass Rename ===

test('prefix and suffix inputs visible after file upload', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  await expect.element(page.getByRole('textbox', { name: /Mass Rename Prefix/i })).toBeVisible()
  await expect.element(page.getByRole('textbox', { name: /Mass Rename Suffix/i })).toBeVisible()
})

test('prefix and suffix are sent in rename_map', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunNoCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('textbox', { name: /Mass Rename Prefix/i }).fill('pre-')
  await page.getByRole('textbox', { name: /Mass Rename Suffix/i }).fill('-suf')
  await page.getByRole('button', { name: 'Analyze' }).click()

  const callArgs = (api.catalogs.import as Mock).mock.calls[0][0]
  expect(callArgs.rename_map).toEqual({
    entity_types: { Server: 'pre-Server-suf' },
    type_definitions: { Status: 'pre-Status-suf' },
  })
})

// === Collision Resolution Step ===

test('collision step shows Continue disabled when unreused collision has no prefix/suffix', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunWithCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Error shown for create-new without prefix/suffix
  await expect.element(page.getByText(/name already exists/i)).toBeVisible()
  // Continue disabled because Status conflict is not reused and no prefix/suffix
  await expect.element(page.getByRole('button', { name: 'Continue' })).toHaveAttribute('disabled')
})

test('collision step shows Continue enabled when no conflicts (only identical)', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Continue should be enabled
  const continueBtn = page.getByRole('button', { name: 'Continue' })
  await expect.element(continueBtn).toBeVisible()
  // Should NOT be disabled
  expect(continueBtn.element().hasAttribute('disabled')).toBe(false)
})

test('identical collision toggle switches between Reuse/Create via checkbox', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Server is auto-reused (identical, entity_type) — checkbox checked with "Reuse existing" label
  const checkbox = page.getByRole('checkbox', { name: 'Reuse existing' })
  await expect.element(checkbox).toBeVisible()
  await expect.element(checkbox).toBeChecked()
  // Uncheck to toggle to Create new
  await checkbox.click()
  await expect.element(page.getByRole('checkbox', { name: 'Create new' })).toBeVisible()
  await expect.element(page.getByRole('checkbox', { name: 'Create new' })).not.toBeChecked()
  // Check again to toggle back
  await page.getByRole('checkbox', { name: 'Create new' }).click()
  await expect.element(page.getByRole('checkbox', { name: 'Reuse existing' })).toBeChecked()
})

test('identical type_definition and V2+ entity_type are auto-reused', async () => {
  const dryRunMixed = {
    status: 'ok',
    collisions: [
      { type: 'type_definition', name: 'hex12', resolution: 'identical', version: 1, detail: 'Same' },
      { type: 'entity_type', name: 'mcp-server', resolution: 'identical', version: 7, detail: 'Same' },
    ],
    summary: { total_entities: 4, conflicts: 0, identical: 2, new: 2 },
  }
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunMixed)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Both should show checked checkboxes (auto-selected for all identical, not just V1 entity_types)
  const checkboxes = page.getByRole('checkbox', { name: 'Reuse existing' })
  expect(checkboxes.elements().length).toBe(2)
})

test('Back button in collision step returns to upload', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await page.getByRole('button', { name: 'Back' }).click()
  // Should be back at upload step
  await expect.element(page.getByText('Catalog File (JSON)')).toBeVisible()
})

test('Continue from collisions goes to confirm step', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await page.getByRole('button', { name: 'Continue' }).click()
  await expect.element(page.getByText(/Ready to import/)).toBeVisible()
})

test('conflict resolved by reuse enables Continue', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunWithCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Continue disabled (Status conflict not reused, no prefix/suffix)
  await expect.element(page.getByRole('button', { name: 'Continue' })).toHaveAttribute('disabled')
  // Check the conflict checkbox to reuse existing
  await page.getByRole('checkbox', { name: 'Create new' }).click()
  // Now both Server and Status are reused — Continue should be enabled
  const continueBtn = page.getByRole('button', { name: 'Continue' })
  expect(continueBtn.element().hasAttribute('disabled')).toBe(false)
})

test('conflict reuse sends reuse_existing for conflict entity', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunWithCollisions)
    .mockResolvedValueOnce(importResult)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Check the conflict checkbox
  await page.getByRole('checkbox', { name: 'Create new' }).click()
  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  const secondCall = (api.catalogs.import as Mock).mock.calls[1]
  expect(secondCall[0].reuse_existing).toEqual(expect.arrayContaining(['Server', 'Status']))
})

test('create new without prefix/suffix shows error text', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Uncheck the auto-reused identical entity
  await page.getByRole('checkbox', { name: 'Reuse existing' }).click()
  // Error text should appear (not just a warning)
  await expect.element(page.getByText(/name already exists/i)).toBeVisible()
  // Continue should be disabled
  await expect.element(page.getByRole('button', { name: 'Continue' })).toHaveAttribute('disabled')
})

test('create new with prefix/suffix shows renamed name and enables Continue', async () => {
  // Use a dry run that returns collisions (identical only, so handleAnalyze goes to collision step)
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  // Set prefix before analyze
  await page.getByRole('textbox', { name: /Mass Rename Prefix/i }).fill('pre-')
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Uncheck to toggle to Create new
  await page.getByRole('checkbox', { name: 'Reuse existing' }).click()
  // Should show the renamed name (with prefix applied)
  await expect.element(page.getByText(/pre-Server/)).toBeVisible()
  // Continue should be enabled (prefix is set)
  const continueBtn = page.getByRole('button', { name: 'Continue' })
  expect(continueBtn.element().hasAttribute('disabled')).toBe(false)
})

test('collision table has Use existing column header', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // PatternFly wraps <Th> text in a child span — check the inner text element
  const header = document.querySelector('th.pf-v6-c-table__th:last-child')
  expect(header?.textContent).toBe('Use existing')
})

// === Confirm Step ===

test('confirm step shows import summary', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunNoCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await expect.element(page.getByText(/3 new entities/)).toBeVisible()
  await expect.element(page.getByRole('button', { name: 'Import' })).toBeVisible()
})

test('confirm step Back goes to upload when no collisions', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunNoCollisions)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  // Now on confirm step
  await page.getByRole('button', { name: 'Back' }).click()
  // Should go back to upload (no collisions, so goes to upload)
  await expect.element(page.getByText('Catalog File (JSON)')).toBeVisible()
})

test('confirm step Back goes to collisions when there were collisions', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  // Now on collisions step — click Continue
  await page.getByRole('button', { name: 'Continue' }).click()
  // Now on confirm step — click Back
  await page.getByRole('button', { name: 'Back' }).click()
  // Should go back to collisions (has collisions)
  await expect.element(page.getByRole('gridcell', { name: 'Server' })).toBeVisible()
})

// === Import Execution ===

test('clicking Import calls API without dry_run', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions) // dry run
    .mockResolvedValueOnce(importResult) // actual import
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  // On confirm step
  await page.getByRole('button', { name: 'Import' }).click()

  // Second call should be the actual import (no dry_run)
  expect(api.catalogs.import).toHaveBeenCalledTimes(2)
  const secondCall = (api.catalogs.import as Mock).mock.calls[1]
  expect(secondCall[0]).toHaveProperty('data')
  expect(secondCall[1]).toBeUndefined() // no dry_run param
})

test('import sends reuse_existing when toggled', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunIdenticalOnly) // dry run
    .mockResolvedValueOnce(importResult) // actual import
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  // Server is auto-reused — keep it as reuse
  await page.getByRole('button', { name: 'Continue' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  const secondCall = (api.catalogs.import as Mock).mock.calls[1]
  expect(secondCall[0].reuse_existing).toEqual(['Server'])
})

test('import error shows error alert', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions) // dry run
    .mockRejectedValueOnce(new Error('500: server error')) // import fails
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  await expect.element(page.getByText('500: server error')).toBeVisible()
})

test('import non-Error exception shows generic message', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions)
    .mockRejectedValueOnce('string error')
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  await expect.element(page.getByText('Import failed')).toBeVisible()
})

// === Done Step ===

test('done step shows success with import details', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions)
    .mockResolvedValueOnce(importResult)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  // Title changes to "Import Complete"
  await expect.element(page.getByText('Import Complete')).toBeVisible()
  // Success alert
  await expect.element(page.getByText(/my-catalog.*imported successfully/)).toBeVisible()
  // Details
  await expect.element(page.getByText(/2 types created/)).toBeVisible()
  await expect.element(page.getByText(/1 reused/)).toBeVisible()
  await expect.element(page.getByText(/5 instances created/)).toBeVisible()
  await expect.element(page.getByText(/3 links created/)).toBeVisible()
  // View Catalog button
  await expect.element(page.getByRole('button', { name: 'View Catalog' })).toBeVisible()
})

test('clicking View Catalog calls onClose and onSuccess', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions)
    .mockResolvedValueOnce(importResult)
  const { props } = renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  await page.getByRole('button', { name: 'View Catalog' }).click()
  expect(props.onClose).toHaveBeenCalled()
  expect(props.onSuccess).toHaveBeenCalledWith('my-catalog')
})

test('closing modal via X on done step calls onClose but not onSuccess', async () => {
  ;(api.catalogs.import as Mock)
    .mockResolvedValueOnce(dryRunNoCollisions)
    .mockResolvedValueOnce(importResult)
  const { props } = renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()
  await page.getByRole('button', { name: 'Import' }).click()

  await expect.element(page.getByText('Import Complete')).toBeVisible()
  await page.getByRole('button', { name: 'Close' }).click()
  // X button should close (triggering list refresh in parent) but NOT navigate
  expect(props.onClose).toHaveBeenCalled()
  expect(props.onSuccess).not.toHaveBeenCalled()
})

// === Collision detail: catalog/catalog_version type shows "Create" not toggle ===

test('catalog type collision shows Create not toggle', async () => {
  const dryRunCatalogCollision = {
    status: 'ok',
    collisions: [
      { type: 'catalog', name: 'my-catalog', resolution: 'identical', detail: 'Same catalog' },
      { type: 'catalog_version', name: 'v1.0', resolution: 'identical', detail: 'Same version' },
    ],
    summary: { total_entities: 2, conflicts: 0, identical: 2, new: 0 },
  }
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunCatalogCollision)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // catalog and catalog_version types should show "Create" text, not toggle button
  const creates = page.getByText('Create', { exact: true })
  expect(creates.elements().length).toBeGreaterThanOrEqual(2)
})

test('catalog/CV conflict rows show error message', async () => {
  const dryRunCatalogConflict = {
    status: 'conflicts_found',
    collisions: [
      { type: 'catalog', name: 'my-catalog', resolution: 'conflict', detail: 'Catalog name already exists' },
      { type: 'catalog_version', name: 'v1.0', resolution: 'conflict', detail: 'CV label already exists' },
    ],
    summary: { total_entities: 2, conflicts: 2, identical: 0, new: 0 },
  }
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunCatalogConflict)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  const errors = page.getByText(/go back and change/i)
  expect(errors.elements().length).toBe(2)
})

test('Continue disabled when catalog name has conflict', async () => {
  const dryRunCatalogConflict = {
    status: 'conflicts_found',
    collisions: [
      { type: 'catalog', name: 'my-catalog', resolution: 'conflict', detail: 'Catalog name already exists' },
      { type: 'entity_type', name: 'Server', resolution: 'identical', version: 1, detail: '' },
    ],
    summary: { total_entities: 2, conflicts: 1, identical: 1, new: 0 },
  }
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunCatalogConflict)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await expect.element(page.getByRole('button', { name: 'Continue' })).toHaveAttribute('disabled')
})

test('Continue disabled when catalog_version label has conflict', async () => {
  const dryRunCVConflict = {
    status: 'conflicts_found',
    collisions: [
      { type: 'catalog_version', name: 'v1.0', resolution: 'conflict', detail: 'CV label already exists' },
    ],
    summary: { total_entities: 1, conflicts: 1, identical: 0, new: 0 },
  }
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunCVConflict)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  await expect.element(page.getByRole('button', { name: 'Continue' })).toHaveAttribute('disabled')
})

// === CV Label editing ===

test('typing in CV Label field updates the label', async () => {
  renderModal()
  await uploadFile(sampleExportData)
  const cvInput = page.getByRole('textbox', { name: /Catalog Version Label/i })
  await expect.element(cvInput).toHaveValue('v1.0')
  // Type a new value to exercise the onChange callback (ImportCatalogModal.tsx:218)
  await cvInput.fill('v2.0-beta')
  await expect.element(cvInput).toHaveValue('v2.0-beta')
})

// === Collision detail: version display ===

test('collision shows version number when available', async () => {
  ;(api.catalogs.import as Mock).mockResolvedValue(dryRunIdenticalOnly)
  renderModal()
  await uploadFile(sampleExportData)
  await page.getByRole('button', { name: 'Analyze' }).click()

  // Should show "identical (V1)" for the collision with version=1
  await expect.element(page.getByText(/identical \(V1\)/)).toBeVisible()
})
