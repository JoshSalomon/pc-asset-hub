import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { usePinManagement } from './usePinManagement'
import type { CatalogVersionPin } from '../types'

vi.mock('../api/client', () => ({
  api: {
    catalogVersions: {
      addPin: vi.fn(),
      updatePin: vi.fn(),
      updatePinDryRun: vi.fn(),
      removePin: vi.fn(),
    },
    versions: { list: vi.fn() },
    entityTypes: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockPin: CatalogVersionPin = {
  pin_id: 'pin-1',
  entity_type_name: 'Model',
  entity_type_id: 'et-1',
  entity_type_version_id: 'etv-1',
  version: 3,
}

function TestComponent({ cvId, loadPins, onError }: {
  cvId?: string
  loadPins: () => void
  onError: (msg: string) => void
}) {
  const pm = usePinManagement({ catalogVersionId: cvId, loadPins, onError })
  return (
    <div>
      <span data-testid="addPinOpen">{String(pm.addPinOpen)}</span>
      <span data-testid="addPinError">{pm.addPinError || ''}</span>
      <span data-testid="entityTypes">{pm.entityTypes.map(et => et.name).join(',')}</span>
      <span data-testid="selectedEtId">{pm.selectedEtId}</span>
      <span data-testid="selectedEtvId">{pm.selectedEtvId}</span>
      <span data-testid="entityTypeVersions">{pm.entityTypeVersions.map(v => `V${v.version}`).join(',')}</span>
      <span data-testid="pinVersionSelectOpen">{pm.pinVersionSelectOpen || ''}</span>
      <span data-testid="pinVersionOptions">{JSON.stringify(pm.pinVersionOptions)}</span>
      <span data-testid="migrationPreview">{pm.migrationPreview ? 'showing' : 'none'}</span>

      <button onClick={pm.handleOpenAddPin}>OpenAddPin</button>
      <button onClick={pm.handleCloseAddPin}>CloseAddPin</button>
      <button onClick={() => pm.handleSelectEntityType('et-1')}>PickEntityType</button>
      <button onClick={() => pm.setSelectedEtvId('etv-1')}>ChooseETV</button>
      <button onClick={pm.handleAddPin}>SubmitAddPin</button>
      <button onClick={() => pm.handleRemovePin('pin-1')}>RemovePin</button>
      <button onClick={() => pm.handleTogglePinVersionSelect(mockPin)}>ToggleVersion</button>
      <button onClick={pm.closePinVersionSelect}>CloseVersionSelect</button>
      <button onClick={() => pm.handleUpdatePinVersion(mockPin, 'etv-new')}>UpdateVersion</button>
    </div>
  )
}

let loadPins: Mock
let onError: Mock

beforeEach(() => {
  vi.clearAllMocks()
  loadPins = vi.fn()
  onError = vi.fn()
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [
    { id: 'et-1', name: 'Model', created_at: '', updated_at: '' },
    { id: 'et-2', name: 'Tool', created_at: '', updated_at: '' },
  ] })
  ;(api.versions.list as Mock).mockResolvedValue({ items: [
    { id: 'etv-1', entity_type_id: 'et-1', version: 1, description: 'V1', created_at: '' },
    { id: 'etv-2', entity_type_id: 'et-1', version: 2, description: 'V2', created_at: '' },
  ] })
  ;(api.catalogVersions.addPin as Mock).mockResolvedValue({})
  ;(api.catalogVersions.updatePin as Mock).mockResolvedValue({})
  ;(api.catalogVersions.updatePinDryRun as Mock).mockResolvedValue({ pin: {}, migration: null })
  ;(api.catalogVersions.removePin as Mock).mockResolvedValue(undefined)
})

test('initial state: modal closed, no errors', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('false')
  await expect.element(page.getByTestId('addPinError')).toHaveTextContent('')
})

test('handleOpenAddPin loads entity types and opens modal', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'OpenAddPin' }).click()
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('true')
  await expect.element(page.getByTestId('entityTypes')).toHaveTextContent('Model,Tool')
})

test('handleCloseAddPin closes modal and clears error', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'OpenAddPin' }).click()
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('true')
  await page.getByRole('button', { name: 'CloseAddPin' }).click()
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('false')
})

test('handleSelectEntityType loads versions', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'PickEntityType' }).click()
  await expect.element(page.getByTestId('selectedEtId')).toHaveTextContent('et-1')
  await expect.element(page.getByTestId('entityTypeVersions')).toHaveTextContent('V1,V2')
})

test('handleSelectEntityType clears versions on error', async () => {
  ;(api.versions.list as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'PickEntityType' }).click()
  await expect.element(page.getByTestId('entityTypeVersions')).toHaveTextContent('')
})

test('handleAddPin calls API and closes modal', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'OpenAddPin' }).click()
  await page.getByRole('button', { name: 'ChooseETV' }).click()
  await page.getByRole('button', { name: 'SubmitAddPin' }).click()
  expect(api.catalogVersions.addPin).toHaveBeenCalledWith('cv1', 'etv-1')
  await vi.waitFor(() => expect(loadPins).toHaveBeenCalled())
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('false')
})

test('handleAddPin does nothing without cvId', async () => {
  render(<TestComponent cvId={undefined} loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'ChooseETV' }).click()
  await page.getByRole('button', { name: 'SubmitAddPin' }).click()
  expect(api.catalogVersions.addPin).not.toHaveBeenCalled()
})

test('handleAddPin does nothing without selectedEtvId', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'SubmitAddPin' }).click()
  expect(api.catalogVersions.addPin).not.toHaveBeenCalled()
})

test('handleAddPin error sets addPinError', async () => {
  ;(api.catalogVersions.addPin as Mock).mockRejectedValue(new Error('409: conflict'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'OpenAddPin' }).click()
  await page.getByRole('button', { name: 'ChooseETV' }).click()
  await page.getByRole('button', { name: 'SubmitAddPin' }).click()
  await expect.element(page.getByTestId('addPinError')).toHaveTextContent('409: conflict')
})

test('handleRemovePin calls API and loadPins', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'RemovePin' }).click()
  expect(api.catalogVersions.removePin).toHaveBeenCalledWith('cv1', 'pin-1')
  await vi.waitFor(() => expect(loadPins).toHaveBeenCalled())
})

test('handleRemovePin does nothing without cvId', async () => {
  render(<TestComponent cvId={undefined} loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'RemovePin' }).click()
  expect(api.catalogVersions.removePin).not.toHaveBeenCalled()
})

test('handleRemovePin error calls onError', async () => {
  ;(api.catalogVersions.removePin as Mock).mockRejectedValue(new Error('500: failed'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'RemovePin' }).click()
  await vi.waitFor(() => expect(onError).toHaveBeenCalledWith('500: failed'))
})

test('handleTogglePinVersionSelect opens and loads versions', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('pin-1')
  // Versions should be loaded for et-1
  await vi.waitFor(() => expect(api.versions.list).toHaveBeenCalledWith('et-1'))
})

test('handleTogglePinVersionSelect toggles closed on second click', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('pin-1')
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('')
})

test('closePinVersionSelect clears open state', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('pin-1')
  await page.getByRole('button', { name: 'CloseVersionSelect' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('')
})

test('handleUpdatePinVersion calls API and loadPins', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  expect(api.catalogVersions.updatePin).toHaveBeenCalledWith('cv1', 'pin-1', 'etv-new')
  await vi.waitFor(() => expect(loadPins).toHaveBeenCalled())
})

test('handleUpdatePinVersion does nothing without cvId', async () => {
  render(<TestComponent cvId={undefined} loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  expect(api.catalogVersions.updatePin).not.toHaveBeenCalled()
})

test('handleUpdatePinVersion error calls onError', async () => {
  ;(api.catalogVersions.updatePinDryRun as Mock).mockRejectedValue(new Error('400: mismatch'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  await vi.waitFor(() => expect(onError).toHaveBeenCalledWith('400: mismatch'))
})

test('handleOpenAddPin handles entityTypes.list error gracefully', async () => {
  ;(api.entityTypes.list as Mock).mockRejectedValue(new Error('network'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'OpenAddPin' }).click()
  // Modal still opens despite error
  await expect.element(page.getByTestId('addPinOpen')).toHaveTextContent('true')
  await expect.element(page.getByTestId('entityTypes')).toHaveTextContent('')
})

test('loadPinVersionOptions uses cache on second toggle (skips API call)', async () => {
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  // First toggle: loads from API
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await vi.waitFor(() => expect(api.versions.list).toHaveBeenCalledTimes(1))
  // Close
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  // Re-open: should use cache, NOT call API again
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  await expect.element(page.getByTestId('pinVersionSelectOpen')).toHaveTextContent('pin-1')
  // Still only 1 API call — cache hit on L78
  expect(api.versions.list).toHaveBeenCalledTimes(1)
})

test('loadPinVersionOptions handles error gracefully', async () => {
  ;(api.versions.list as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'ToggleVersion' }).click()
  // Should not crash, pinVersionOptions should have empty array for et-1
  await vi.waitFor(() => {
    const text = page.getByTestId('pinVersionOptions').element().textContent || ''
    expect(text).toContain('"et-1":[]')
  })
})

// Bug fix: clean remap (all attrs match, no warnings) should NOT show dialog
test('handleUpdatePinVersion skips dialog for clean remap with no warnings', async () => {
  ;(api.catalogVersions.updatePinDryRun as Mock).mockResolvedValue({
    pin: {},
    migration: {
      affected_catalogs: 1,
      affected_instances: 5,
      attribute_mappings: [
        { old_name: 'hostname', new_name: 'hostname', action: 'remap' },
        { old_name: 'port', new_name: 'port', action: 'remap' },
      ],
      warnings: [],
    },
  })
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  // Should NOT show migration preview — all clean remaps
  await expect.element(page.getByTestId('migrationPreview')).toHaveTextContent('none')
  // Should apply directly
  await vi.waitFor(() => expect(api.catalogVersions.updatePin).toHaveBeenCalledWith('cv1', 'pin-1', 'etv-new'))
})

// Dialog SHOULD show when there are non-remap actions
test('handleUpdatePinVersion shows dialog when orphaned attr exists', async () => {
  ;(api.catalogVersions.updatePinDryRun as Mock).mockResolvedValue({
    pin: {},
    migration: {
      affected_catalogs: 1,
      affected_instances: 5,
      attribute_mappings: [
        { old_name: 'hostname', new_name: 'hostname', action: 'remap' },
        { old_name: 'old_field', action: 'orphaned' },
      ],
      warnings: [{ type: 'deleted_attribute', attribute: 'old_field', affected_instances: 5 }],
    },
  })
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  // Should show migration preview — has orphaned attr
  await expect.element(page.getByTestId('migrationPreview')).toHaveTextContent('showing')
  // Should NOT have called updatePin (waiting for confirm)
  expect(api.catalogVersions.updatePin).not.toHaveBeenCalled()
})

// Bug fix: dialog should NOT show when warnings exist but 0 instances affected
test('handleUpdatePinVersion skips dialog when warnings but 0 affected instances', async () => {
  ;(api.catalogVersions.updatePinDryRun as Mock).mockResolvedValue({
    pin: {},
    migration: {
      affected_catalogs: 0,
      affected_instances: 0,
      attribute_mappings: [
        { old_name: 'dd', new_name: 'd', action: 'remap' },
      ],
      warnings: [
        { type: 'renamed', attribute: 'd', affected_instances: 0, old_type: 'dd', new_type: 'd' },
      ],
    },
  })
  render(<TestComponent cvId="cv1" loadPins={loadPins} onError={onError} />)
  await page.getByRole('button', { name: 'UpdateVersion' }).click()
  // Should NOT show dialog — 0 instances affected, nothing to migrate
  await expect.element(page.getByTestId('migrationPreview')).toHaveTextContent('none')
  // Should apply directly
  await vi.waitFor(() => expect(api.catalogVersions.updatePin).toHaveBeenCalledWith('cv1', 'pin-1', 'etv-new'))
})
