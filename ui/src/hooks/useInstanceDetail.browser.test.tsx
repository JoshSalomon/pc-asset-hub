import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api, setAuthRole } from '../api/client'
import { useInstanceDetail } from './useInstanceDetail'
import type { SnapshotAssociation, EntityInstance } from '../types'

vi.mock('../api/client', () => ({
  api: {
    instances: { get: vi.fn(), listContained: vi.fn() },
    links: { forwardRefs: vi.fn(), reverseRefs: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const schemaAssocs: SnapshotAssociation[] = [
  {
    id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing',
    target_entity_type_id: 'et2', target_entity_type_name: 'tool',
    source_role: '', target_role: '', source_cardinality: '', target_cardinality: '',
  },
]

const mockInstance: EntityInstance = {
  id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'inst-a', description: 'First',
  version: 1, attributes: [], created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockInstanceWithParent: EntityInstance = {
  ...mockInstance, id: 'i1-with-parent', parent_instance_id: 'parent1',
}

function TestComponent({ catalogName, entityTypeName, assocs, role }: { catalogName?: string; entityTypeName: string; assocs?: SnapshotAssociation[]; role?: 'RO' | 'RW' | 'Admin' | 'SuperAdmin' }) {
  const detail = useInstanceDetail(catalogName, entityTypeName, assocs || schemaAssocs, role || 'Admin')
  return (
    <div>
      <span data-testid="selected">{detail.selectedInstance?.name || ''}</span>
      <span data-testid="parent-name">{detail.parentName}</span>
      <span data-testid="children-count">{detail.children.length}</span>
      <span data-testid="children-loading">{String(detail.childrenLoading)}</span>
      <span data-testid="fwd-refs-count">{detail.forwardRefs.length}</span>
      <span data-testid="rev-refs-count">{detail.reverseRefs.length}</span>
      <span data-testid="refs-loading">{String(detail.refsLoading)}</span>
      <button data-testid="select" onClick={() => detail.selectInstance(mockInstance.id)}>SelectInst</button>
      <button data-testid="select-parent" onClick={() => detail.selectInstance(mockInstanceWithParent.id)}>SelectParent</button>
      <button data-testid="select-null" onClick={() => detail.selectInstance(null)}>SelectNull</button>
      <button data-testid="clear" onClick={detail.clearSelection}>ClearSel</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'parent1') return Promise.resolve({ id: 'parent1', name: 'parent-inst' })
    if (id === 'i1') return Promise.resolve(mockInstance)
    if (id === mockInstanceWithParent.id) return Promise.resolve(mockInstanceWithParent)
    return Promise.resolve({ id, name: `inst-${id}` })
  })
  ;(api.instances.listContained as Mock).mockResolvedValue({ items: [{ id: 'c1', name: 'child-a' }], total: 1 })
  ;(api.links.forwardRefs as Mock).mockResolvedValue([{ link_id: 'l1', association_name: 'uses', association_type: 'directional', instance_id: 'i2', instance_name: 'target', entity_type_name: 'tool' }])
  ;(api.links.reverseRefs as Mock).mockResolvedValue([{ link_id: 'l2', association_name: 'dep', association_type: 'directional', instance_id: 'i3', instance_name: 'source', entity_type_name: 'server' }])
})

// T-19.20: selectInstance loads parent name, children, refs
test('T-19.20: useInstanceDetail selectInstance loads data', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select-parent').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  await expect.element(page.getByTestId('parent-name')).toHaveTextContent('parent-inst')
  await expect.element(page.getByTestId('children-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('fwd-refs-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('rev-refs-count')).toHaveTextContent('1')
})

// T-19.21: selectInstance with no parent skips parent name load
test('T-19.21: useInstanceDetail skips parent name when no parent', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  // Wait for children to load to be sure all async work is done
  await expect.element(page.getByTestId('children-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('parent-name')).toHaveTextContent('')
  // instances.get is called once for re-fetch, but NOT again for parent resolution
  expect(api.instances.get).toHaveBeenCalledTimes(1)
  expect(api.instances.get).toHaveBeenCalledWith('my-catalog', 'model', 'i1')
})

// T-19.22: selectInstance handles parent name load error (falls back to ID)
test('T-19.22: useInstanceDetail parent name error falls back to ID', async () => {
  // First call (re-fetch) succeeds, second call (parent resolution) fails
  ;(api.instances.get as Mock).mockImplementation((_cat: string, _et: string, id: string) => {
    if (id === 'i1-with-parent') return Promise.resolve(mockInstanceWithParent)
    return Promise.reject(new Error('Not found'))
  })
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select-parent').click()
  await expect.element(page.getByTestId('parent-name')).toHaveTextContent('parent1')
})

// T-19.23: selectInstance handles children load error
test('T-19.23: useInstanceDetail children load error', async () => {
  ;(api.instances.listContained as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('children-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('children-count')).toHaveTextContent('0')
})

// T-19.24: selectInstance handles refs load error
test('T-19.24: useInstanceDetail refs load error', async () => {
  ;(api.links.forwardRefs as Mock).mockRejectedValue(new Error('fail'))
  ;(api.links.reverseRefs as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('refs-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('fwd-refs-count')).toHaveTextContent('0')
  await expect.element(page.getByTestId('rev-refs-count')).toHaveTextContent('0')
})

// T-19.25: clearSelection resets all detail state
test('T-19.25: useInstanceDetail clearSelection resets state', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  await page.getByTestId('clear').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('')
  await expect.element(page.getByTestId('children-count')).toHaveTextContent('0')
})

// T-19.26: selectInstance with null clears selection
test('T-19.26: useInstanceDetail selectInstance null clears', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  await page.getByTestId('select-null').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('')
})

// TD-49: selectInstance calls setAuthRole before making API calls
test('TD-49: selectInstance calls setAuthRole with current role', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" role="RW" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  expect(setAuthRole).toHaveBeenCalledWith('RW')
})

// TD-50: selectInstance re-fetches instance by ID instead of using stale object
test('TD-50: selectInstance fetches instance by ID from API', async () => {
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  // selectInstance('i1') should call api.instances.get to fetch the instance
  expect(api.instances.get).toHaveBeenCalledWith('my-catalog', 'model', 'i1')
})

// Lines 30-34: api.instances.get rejects — clears all state and returns early
test('selectInstance clears state when api.instances.get rejects', async () => {
  // First select an instance so state is populated
  render(<TestComponent catalogName="my-catalog" entityTypeName="model" />)
  await page.getByTestId('select').click()
  await expect.element(page.getByTestId('selected')).toHaveTextContent('inst-a')
  await expect.element(page.getByTestId('children-count')).toHaveTextContent('1')

  // Now make api.instances.get reject for ALL calls
  ;(api.instances.get as Mock).mockRejectedValue(new Error('Instance not found'))

  // Re-select to trigger the error path (lines 30-34)
  await page.getByTestId('select').click()

  // State should be cleared by the catch block
  await expect.element(page.getByTestId('selected')).toHaveTextContent('')
  await expect.element(page.getByTestId('children-count')).toHaveTextContent('0')
  await expect.element(page.getByTestId('fwd-refs-count')).toHaveTextContent('0')
  await expect.element(page.getByTestId('rev-refs-count')).toHaveTextContent('0')
})

// Line 58: outer catch on containment children loading — setChildren([])
// UNREACHABLE: The outer try block (lines 46-58) wraps Array.filter() and
// a for loop with its own inner try/catch. Array.filter() on schemaAssocs
// (always an array) cannot throw, and setChildren (a React state setter)
// cannot throw. The inner try/catch (line 51-54) catches per-iteration API
// errors. No code path in the outer try can throw past the inner catch.
