import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useCatalogDiagram } from './useCatalogDiagram'

vi.mock('../api/client', () => ({
  api: {
    catalogVersions: { listPins: vi.fn() },
    versions: { snapshot: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockPins = [
  { pin_id: 'pin-1', entity_type_name: 'Server', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 2 },
  { pin_id: 'pin-2', entity_type_name: 'Tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
]

const mockSnapshotServer = {
  entity_type: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
  version: { id: 'etv1', version: 2 },
  attributes: [
    { id: 'a1', name: 'hostname', description: '', type: 'string', ordinal: 1, required: false },
  ],
  associations: [
    { id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'Tool', source_role: '', target_role: '', source_cardinality: '1', target_cardinality: '0..n' },
  ],
}

const mockSnapshotTool = {
  entity_type: { id: 'et2', name: 'Tool', created_at: '', updated_at: '' },
  version: { id: 'etv2', version: 1 },
  attributes: [],
  associations: [],
}

function TestComponent({ cvId, trigger }: { cvId?: string; trigger?: boolean }) {
  const { diagramData, diagramLoading, diagramError, loadDiagram } = useCatalogDiagram(cvId)
  return (
    <div>
      <span data-testid="loading">{String(diagramLoading)}</span>
      <span data-testid="count">{diagramData.length}</span>
      <span data-testid="error">{diagramError || ''}</span>
      <span data-testid="names">{diagramData.map(d => d.entityType.name).join(',')}</span>
      {trigger && <button onClick={loadDiagram}>Load</button>}
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

test('returns empty diagramData and loading=false initially', async () => {
  render(<TestComponent cvId="cv1" />)
  await expect.element(page.getByTestId('loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('count')).toHaveTextContent('0')
})

test('loads pins and snapshots when loadDiagram is called', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins })
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et1') return Promise.resolve(mockSnapshotServer)
    return Promise.resolve(mockSnapshotTool)
  })

  render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()

  await expect.element(page.getByTestId('count')).toHaveTextContent('2')
  await expect.element(page.getByTestId('names')).toHaveTextContent('Server,Tool')
  expect(api.catalogVersions.listPins).toHaveBeenCalledWith('cv1')
  expect(api.versions.snapshot).toHaveBeenCalledWith('et1', 2)
  expect(api.versions.snapshot).toHaveBeenCalledWith('et2', 1)
})

test('sets diagramLoading=true during fetch', async () => {
  let resolveListPins: (v: any) => void
  ;(api.catalogVersions.listPins as Mock).mockReturnValue(new Promise(r => { resolveListPins = r }))

  render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()
  await expect.element(page.getByTestId('loading')).toHaveTextContent('true')

  // Resolve to complete
  resolveListPins!({ items: [] })
  await expect.element(page.getByTestId('loading')).toHaveTextContent('false')
})

test('does not re-fetch if diagramData is already loaded', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins })
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et1') return Promise.resolve(mockSnapshotServer)
    return Promise.resolve(mockSnapshotTool)
  })

  render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()
  await expect.element(page.getByTestId('count')).toHaveTextContent('2')

  // Click again — should not re-fetch
  await page.getByRole('button', { name: 'Load' }).click()
  expect(api.catalogVersions.listPins).toHaveBeenCalledTimes(1)
})

test('handles API error gracefully — sets error, clears loading', async () => {
  ;(api.catalogVersions.listPins as Mock).mockRejectedValue(new Error('Network error'))

  render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()

  await expect.element(page.getByTestId('loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('error')).toHaveTextContent('Network error')
})

test('handles empty pins list — returns empty diagramData', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: [] })

  render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()

  await expect.element(page.getByTestId('count')).toHaveTextContent('0')
  expect(api.versions.snapshot).not.toHaveBeenCalled()
})

test('does nothing when cvId is undefined', async () => {
  render(<TestComponent cvId={undefined} trigger />)
  await page.getByRole('button', { name: 'Load' }).click()

  expect(api.catalogVersions.listPins).not.toHaveBeenCalled()
  await expect.element(page.getByTestId('count')).toHaveTextContent('0')
})

test('reloads when catalogVersionId changes after successful load', async () => {
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins })
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et1') return Promise.resolve(mockSnapshotServer)
    return Promise.resolve(mockSnapshotTool)
  })

  const { rerender } = await render(<TestComponent cvId="cv1" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()
  await expect.element(page.getByTestId('count')).toHaveTextContent('2')
  expect(api.catalogVersions.listPins).toHaveBeenCalledTimes(1)

  // Change cvId — should allow reloading
  rerender(<TestComponent cvId="cv2" trigger />)
  await page.getByRole('button', { name: 'Load' }).click()
  await vi.waitFor(() => {
    expect(api.catalogVersions.listPins).toHaveBeenCalledTimes(2)
  })
  expect(api.catalogVersions.listPins).toHaveBeenCalledWith('cv2')
})
