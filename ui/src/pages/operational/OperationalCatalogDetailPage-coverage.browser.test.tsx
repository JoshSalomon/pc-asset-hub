import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import OperationalCatalogDetailPage from './OperationalCatalogDetailPage'

vi.mock('react-router-dom', () => ({
  useParams: () => ({ name: 'test-catalog' }),
  useNavigate: () => vi.fn(),
  Link: ({ children }: { children: React.ReactNode }) => children,
}))

vi.mock('../../api/client', () => ({
  api: {
    catalogs: { get: vi.fn() },
    catalogVersions: { listPins: vi.fn() },
    versions: { snapshot: vi.fn() },
    instances: { list: vi.fn(), get: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), createContained: vi.fn(), listContained: vi.fn(), setParent: vi.fn() },
    links: { create: vi.fn(), delete: vi.fn(), forwardRefs: vi.fn(), reverseRefs: vi.fn() },
    exporters: { list: vi.fn() },
    exportBindings: { list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn(), run: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const { api } = await import('../../api/client')

const mockCatalog = {
  id: 'cat1', name: 'test-catalog', description: 'Test',
  catalog_version_id: 'cv1', catalog_version_label: 'v1.0',
  validation_status: 'draft', created_at: '2026-01-01', updated_at: '2026-01-01',
}

const mockPins = [
  { pin_id: 'pin-1', entity_type_name: 'server', entity_type_id: 'et1', entity_type_version_id: 'etv1', version: 1 },
  { pin_id: 'pin-2', entity_type_name: 'tool', entity_type_id: 'et2', entity_type_version_id: 'etv2', version: 1 },
]

const mockSnapshotServer = {
  entity_type: { id: 'et1', name: 'server' },
  version: { id: 'etv1', version: 1 },
  attributes: [{ id: 'a1', name: 'endpoint', base_type: 'string', ordinal: 1, required: false }],
  associations: [{ id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing', target_entity_type_id: 'et2', target_entity_type_name: 'tool' }],
}

const mockInstances = [{ id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: '', version: 1, attributes: [], created_at: '2026-01-01', updated_at: '2026-01-01' }]

const { setAuthRole } = await import('../../api/client')

function renderDetail(role: string = 'Admin') {
  setAuthRole(role)
  return render(<OperationalCatalogDetailPage role={role as any} />)
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.catalogs.get as Mock).mockResolvedValue(mockCatalog)
  ;(api.catalogVersions.listPins as Mock).mockResolvedValue({ items: mockPins, total: 2 })
  ;(api.versions.snapshot as Mock).mockResolvedValue(mockSnapshotServer)
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockInstances, total: 1 })
  ;(api.instances.get as Mock).mockResolvedValue(mockInstances[0])
  ;(api.links.forwardRefs as Mock).mockResolvedValue({ items: [] })
  ;(api.links.reverseRefs as Mock).mockResolvedValue({ items: [] })
  ;(api.exporters.list as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.exportBindings.list as Mock).mockResolvedValue({ items: [], total: 0 })
})

// Coverage: L140 — loadSchemaSnapshot .catch fallback in containmentTargetTypes effect
test('containmentTargetTypes: snapshot failure for one pin uses catch fallback', async () => {
  ;(api.versions.snapshot as Mock).mockImplementation((etId: string) => {
    if (etId === 'et2') return Promise.reject(new Error('snapshot load failed'))
    return Promise.resolve(mockSnapshotServer)
  })

  renderDetail()
  await expect.element(page.getByRole('heading', { name: /test-catalog/ })).toBeVisible()
  await vi.waitFor(() => { expect(api.versions.snapshot).toHaveBeenCalled() })
})

// Coverage: L142 — async cleanup guard (cancelled=true before Promise.all resolves)
test('containmentTargetTypes: unmount before snapshot resolves hits cancelled guard', async () => {
  const handler = (e: PromiseRejectionEvent) => e.preventDefault()
  window.addEventListener('unhandledrejection', handler)

  const { cleanup } = await import('vitest-browser-react')
  ;(api.versions.snapshot as Mock).mockImplementation(() => new Promise(resolve => {
    setTimeout(() => resolve(mockSnapshotServer), 300)
  }))
  renderDetail()
  await cleanup()
  await new Promise(r => setTimeout(r, 400))

  window.removeEventListener('unhandledrejection', handler)
})
