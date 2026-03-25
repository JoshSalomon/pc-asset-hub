import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useInstances } from './useInstances'
import type { SnapshotAttribute, EntityInstance } from '../types'

vi.mock('../api/client', () => ({
  api: {
    instances: { list: vi.fn(), create: vi.fn(), update: vi.fn(), delete: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const schemaAttrs: SnapshotAttribute[] = [
  { id: 'sys-name', name: 'name', type: 'string', description: '', ordinal: -2, required: true, system: true },
  { id: 'sys-desc', name: 'description', type: 'string', description: '', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'port', type: 'number', description: '', ordinal: 1, required: false },
  { id: 'a2', name: 'hostname', type: 'string', description: '', ordinal: 2, required: false },
]

const mockInstances: EntityInstance[] = [
  {
    id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
    version: 1, attributes: [
      { name: 'name', type: 'string', value: 'inst-a', system: true },
      { name: 'description', type: 'string', value: 'First', system: true },
      { name: 'port', type: 'number', value: 8080 },
      { name: 'hostname', type: 'string', value: 'host-a' },
    ],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  },
]

function TestComponent({ catalogName, entityTypeName }: { catalogName?: string; entityTypeName: string }) {
  const inst = useInstances(catalogName, entityTypeName, schemaAttrs, 'Admin')
  return (
    <div>
      <span data-testid="inst-count">{inst.instances.length}</span>
      <span data-testid="inst-total">{inst.instTotal}</span>
      <span data-testid="inst-loading">{String(inst.instLoading)}</span>
      <span data-testid="create-open">{String(inst.createOpen)}</span>
      <span data-testid="create-error">{inst.createError || ''}</span>
      <span data-testid="edit-target">{inst.editTarget?.name || ''}</span>
      <span data-testid="edit-error">{inst.editError || ''}</span>
      <span data-testid="delete-target">{inst.deleteTarget?.name || ''}</span>
      <span data-testid="delete-error">{inst.deleteError || ''}</span>
      <button data-testid="btn-open-create" onClick={inst.openCreate}>OpenCreate</button>
      <button data-testid="btn-open-edit" onClick={() => inst.openEdit(mockInstances[0])}>OpenEdit</button>
      <button data-testid="btn-open-delete" onClick={() => inst.openDelete(mockInstances[0])}>OpenDelete</button>
      <button data-testid="btn-create" onClick={() => inst.handleCreate('new-inst', 'desc', {})}>DoCreate</button>
      <button data-testid="btn-create-with-attrs" onClick={() => inst.handleCreate('new-inst', 'desc', { port: '9090' })}>DoCreateAttrs</button>
      <button data-testid="btn-edit" onClick={() => inst.handleEdit(1, 'inst-a', 'First', { hostname: 'host-a' })}>DoEdit</button>
      <button data-testid="btn-delete" onClick={inst.handleDelete}>DoDelete</button>
      <button data-testid="btn-load" onClick={inst.loadInstances}>LoadInst</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.instances.list as Mock).mockResolvedValue({ items: mockInstances, total: 1 })
  ;(api.instances.create as Mock).mockResolvedValue({ id: 'i2', name: 'new-inst' })
  ;(api.instances.update as Mock).mockResolvedValue({ id: 'i1', name: 'inst-a', version: 2 })
  ;(api.instances.delete as Mock).mockResolvedValue(undefined)
})

// T-19.09: Loads instances for active entity type
test('T-19.09: useInstances loads instances', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-load').click()
  await expect.element(page.getByTestId('inst-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('inst-total')).toHaveTextContent('1')
})

// T-19.10: Returns early when catalogName is undefined
test('T-19.10: useInstances returns early when catalogName undefined', async () => {
  render(<TestComponent catalogName={undefined} entityTypeName="model" />)
  await page.getByTestId('btn-load').click()
  expect(api.instances.list).not.toHaveBeenCalled()
})

// T-19.11: handleCreate calls API with name, description, attributes
test('T-19.11: useInstances handleCreate calls API', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-create').click()
  await page.getByTestId('btn-create').click()
  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({ name: 'new-inst' }))
})

// T-19.12: handleCreate with number attribute passes parseFloat value
test('T-19.12: useInstances handleCreate with number attr', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-create').click()
  await page.getByTestId('btn-create-with-attrs').click()
  expect(api.instances.create).toHaveBeenCalledWith('my-catalog', 'model', expect.objectContaining({
    attributes: expect.objectContaining({ port: 9090 }),
  }))
})

// T-19.13: handleCreate error sets createError
test('T-19.13: useInstances handleCreate error', async () => {
  ;(api.instances.create as Mock).mockRejectedValue(new Error('Duplicate'))
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-create').click()
  await page.getByTestId('btn-create').click()
  await expect.element(page.getByTestId('create-error')).toHaveTextContent('Duplicate')
})

// T-19.14: handleEdit calls API with version, changed fields
test('T-19.14: useInstances handleEdit calls API', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-edit').click()
  await page.getByTestId('btn-edit').click()
  expect(api.instances.update).toHaveBeenCalledWith('my-catalog', 'model', 'i1', expect.objectContaining({ version: 1 }))
})

// T-19.15: handleEdit error sets editError
test('T-19.15: useInstances handleEdit error', async () => {
  ;(api.instances.update as Mock).mockRejectedValue(new Error('Conflict'))
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-edit').click()
  await page.getByTestId('btn-edit').click()
  await expect.element(page.getByTestId('edit-error')).toHaveTextContent('Conflict')
})

// T-19.16: handleDelete calls API and refreshes list
test('T-19.16: useInstances handleDelete calls API', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-delete').click()
  await page.getByTestId('btn-delete').click()
  expect(api.instances.delete).toHaveBeenCalledWith('my-catalog', 'model', 'i1')
})

// T-19.17: handleDelete error sets deleteError
test('T-19.17: useInstances handleDelete error', async () => {
  ;(api.instances.delete as Mock).mockRejectedValue(new Error('Not found'))
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-delete').click()
  await page.getByTestId('btn-delete').click()
  await expect.element(page.getByTestId('delete-error')).toHaveTextContent('Not found')
})

// T-19.18: openCreate resets form state
test('T-19.18: useInstances openCreate resets form', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-create').click()
  await expect.element(page.getByTestId('create-open')).toHaveTextContent('true')
})

// T-19.19: openEdit populates editTarget
test('T-19.19: useInstances openEdit sets editTarget', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('btn-open-edit').click()
  await expect.element(page.getByTestId('edit-target')).toHaveTextContent('inst-a')
})
