import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useContainmentTree } from './useContainmentTree'
import type { TreeNodeResponse } from '../types'

vi.mock('../api/client', () => ({
  api: {
    instances: { tree: vi.fn(), get: vi.fn() },
    links: { forwardRefs: vi.fn(), reverseRefs: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockTree: TreeNodeResponse[] = [
  {
    instance_id: 'i1', instance_name: 'server-a', entity_type_name: 'server',
    description: 'A server', children: [
      { instance_id: 'i2', instance_name: 'tool-a', entity_type_name: 'tool',
        description: 'A tool', children: [] },
    ],
  },
  { instance_id: 'i3', instance_name: 'server-b', entity_type_name: 'server',
    description: '', children: [] },
]

const mockInstanceDetail = {
  id: 'i1', entity_type_id: 'et1', catalog_id: 'cat1', name: 'server-a',
  description: 'A server', version: 1, attributes: [],
  parent_chain: [],
  created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
}

const mockForwardRefs = [
  { link_id: 'l1', association_name: 'uses', association_type: 'directional',
    instance_id: 'i4', instance_name: 'target-1', entity_type_name: 'model' },
]

const mockReverseRefs = [
  { link_id: 'l2', association_name: 'dep', association_type: 'directional',
    instance_id: 'i5', instance_name: 'source-1', entity_type_name: 'guardrail' },
]

function TestComponent({ catalogName }: { catalogName?: string }) {
  const ct = useContainmentTree(catalogName)
  return (
    <div>
      <span data-testid="tree-count">{ct.tree.length}</span>
      <span data-testid="tree-loading">{String(ct.treeLoading)}</span>
      <span data-testid="selected-name">{ct.selectedInstance?.name || ''}</span>
      <span data-testid="selected-node-id">{ct.selectedNodeId || ''}</span>
      <span data-testid="detail-loading">{String(ct.detailLoading)}</span>
      <span data-testid="fwd-refs-count">{ct.forwardRefs.length}</span>
      <span data-testid="rev-refs-count">{ct.reverseRefs.length}</span>
      <span data-testid="refs-loading">{String(ct.refsLoading)}</span>
      <span data-testid="expanded-count">{ct.expandedNodes.size}</span>
      <button data-testid="load-tree" onClick={ct.loadTree}>LoadTree</button>
      <button data-testid="toggle-i1" onClick={() => ct.toggleNode('i1')}>ToggleI1</button>
      <button data-testid="toggle-group" onClick={() => ct.toggleNode('__group__server')}>ToggleGroup</button>
      <button data-testid="expand-group" onClick={() => ct.expandNode('__group__server')}>ExpandGroup</button>
      <button data-testid="select-node" onClick={() => ct.selectTreeNode(mockTree[0])}>SelectNode</button>
      <button data-testid="navigate-i2" onClick={() => ct.navigateToTreeNode('i2')}>NavigateI2</button>
      <button data-testid="navigate-missing" onClick={() => ct.navigateToTreeNode('missing')}>NavigateMissing</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.instances.tree as Mock).mockResolvedValue(mockTree)
  ;(api.instances.get as Mock).mockResolvedValue(mockInstanceDetail)
  ;(api.links.forwardRefs as Mock).mockResolvedValue(mockForwardRefs)
  ;(api.links.reverseRefs as Mock).mockResolvedValue(mockReverseRefs)
})

// T-20.01: loadTree fetches tree data
test('T-20.01: loadTree fetches and sets tree data', async () => {
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('2')
  expect(api.instances.tree).toHaveBeenCalledWith('my-catalog')
})

// T-20.02: loadTree with undefined catalogName does nothing
test('T-20.02: loadTree with undefined catalogName is no-op', async () => {
  render(<TestComponent catalogName={undefined} />)
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('0')
  expect(api.instances.tree).not.toHaveBeenCalled()
})

// T-20.03: loadTree handles API error gracefully
test('T-20.03: loadTree handles API error', async () => {
  ;(api.instances.tree as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('0')
})

// T-20.04: toggleNode adds/removes from expanded set
test('T-20.04: toggleNode toggles expanded state', async () => {
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('toggle-i1').click()
  await expect.element(page.getByTestId('expanded-count')).toHaveTextContent('1')
  await page.getByTestId('toggle-i1').click()
  await expect.element(page.getByTestId('expanded-count')).toHaveTextContent('0')
})

// T-20.05: selectTreeNode loads instance detail and refs
test('T-20.05: selectTreeNode loads instance and refs', async () => {
  render(<TestComponent catalogName="my-catalog" />)
  // First load tree so navigateToTreeNode works
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('2')
  await page.getByTestId('select-node').click()
  await expect.element(page.getByTestId('selected-name')).toHaveTextContent('server-a')
  await expect.element(page.getByTestId('selected-node-id')).toHaveTextContent('i1')
  await expect.element(page.getByTestId('fwd-refs-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('rev-refs-count')).toHaveTextContent('1')
  expect(api.instances.get).toHaveBeenCalledWith('my-catalog', 'server', 'i1')
  expect(api.links.forwardRefs).toHaveBeenCalledWith('my-catalog', 'server', 'i1')
  expect(api.links.reverseRefs).toHaveBeenCalledWith('my-catalog', 'server', 'i1')
})

// T-20.06: selectTreeNode handles instance get error
test('T-20.06: selectTreeNode handles get error', async () => {
  ;(api.instances.get as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('select-node').click()
  await expect.element(page.getByTestId('detail-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('selected-name')).toHaveTextContent('')
})

// T-20.07: selectTreeNode handles refs error
test('T-20.07: selectTreeNode handles refs error', async () => {
  ;(api.links.forwardRefs as Mock).mockRejectedValue(new Error('fail'))
  ;(api.links.reverseRefs as Mock).mockRejectedValue(new Error('fail'))
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('select-node').click()
  await expect.element(page.getByTestId('refs-loading')).toHaveTextContent('false')
  await expect.element(page.getByTestId('fwd-refs-count')).toHaveTextContent('0')
  await expect.element(page.getByTestId('rev-refs-count')).toHaveTextContent('0')
})

// T-20.08: navigateToTreeNode finds nested node and expands parents
test('T-20.08: navigateToTreeNode finds nested node and expands parents', async () => {
  const toolDetail = {
    id: 'i2', entity_type_id: 'et2', catalog_id: 'cat1', name: 'tool-a',
    description: 'A tool', version: 1, attributes: [], parent_chain: [],
    created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
  }
  ;(api.instances.get as Mock).mockResolvedValue(toolDetail)
  render(<TestComponent catalogName="my-catalog" />)
  // Load tree first
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('2')
  await page.getByTestId('navigate-i2').click()
  // The parent node i1 should be expanded
  await expect.element(page.getByTestId('selected-name')).toHaveTextContent('tool-a')
  // expanded should contain i1 (parent)
  await expect.element(page.getByTestId('expanded-count')).not.toHaveTextContent('0')
})

// T-20.09: navigateToTreeNode with missing node does nothing
test('T-20.09: navigateToTreeNode with missing node is no-op', async () => {
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('2')
  await page.getByTestId('navigate-missing').click()
  await expect.element(page.getByTestId('selected-name')).toHaveTextContent('')
})

// T-20.10: expandNode only adds, never removes
test('T-20.10: expandNode adds without toggling', async () => {
  render(<TestComponent catalogName="my-catalog" />)
  // Expand once
  await page.getByTestId('expand-group').click()
  await expect.element(page.getByTestId('expanded-count')).toHaveTextContent('1')
  // Expand again — should stay expanded (idempotent)
  await page.getByTestId('expand-group').click()
  await expect.element(page.getByTestId('expanded-count')).toHaveTextContent('1')
})

// T-20.11: loadTree returns null/undefined gracefully
test('T-20.11: loadTree handles null response', async () => {
  ;(api.instances.tree as Mock).mockResolvedValue(null)
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByTestId('load-tree').click()
  await expect.element(page.getByTestId('tree-count')).toHaveTextContent('0')
})
