import { useState, useCallback, useMemo } from 'react'
import { api } from '../api/client'
import type { TreeNodeResponse, EntityInstance, ReferenceDetail } from '../types'

function flattenNodes(nodes: TreeNodeResponse[]): TreeNodeResponse[] {
  return nodes.flatMap(n => [n, ...flattenNodes(n.children || [])])
}

export function useContainmentTree(catalogName: string | undefined) {
  const [tree, setTree] = useState<TreeNodeResponse[]>([])
  const [treeLoading, setTreeLoading] = useState(false)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)
  const [selectedTypeName, setSelectedTypeName] = useState<string | null>(null)

  const [selectedInstance, setSelectedInstance] = useState<EntityInstance | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [forwardRefs, setForwardRefs] = useState<ReferenceDetail[]>([])
  const [reverseRefs, setReverseRefs] = useState<ReferenceDetail[]>([])
  const [refsLoading, setRefsLoading] = useState(false)

  const loadTree = useCallback(async (): Promise<TreeNodeResponse[]> => {
    if (!catalogName) return []
    setTreeLoading(true)
    try {
      const t = await api.instances.tree(catalogName)
      const result = t || []
      setTree(result)
      return result
    } catch {
      setTree([])
      return []
    } finally {
      setTreeLoading(false)
    }
  }, [catalogName])

  const selectTreeNode = useCallback(async (node: TreeNodeResponse) => {
    setSelectedNodeId(node.instance_id)
    setSelectedTypeName(node.entity_type_name)
    setDetailLoading(true)
    try {
      const inst = await api.instances.get(catalogName!, node.entity_type_name, node.instance_id)
      setSelectedInstance(inst)
    } catch {
      setSelectedInstance(null)
    } finally {
      setDetailLoading(false)
    }

    setRefsLoading(true)
    try {
      const [fwd, rev] = await Promise.all([
        api.links.forwardRefs(catalogName!, node.entity_type_name, node.instance_id),
        api.links.reverseRefs(catalogName!, node.entity_type_name, node.instance_id),
      ])
      setForwardRefs(fwd || [])
      setReverseRefs(rev || [])
    } catch {
      setForwardRefs([])
      setReverseRefs([])
    } finally {
      setRefsLoading(false)
    }
  }, [catalogName])

  const selectNodeById = useCallback(async (instanceId: string, freshTree: TreeNodeResponse[]) => {
    const node = flattenNodes(freshTree).find(n => n.instance_id === instanceId)
    if (node) await selectTreeNode(node)
  }, [selectTreeNode])

  const findParentNode = useCallback((instanceId: string, nodes: TreeNodeResponse[]): TreeNodeResponse | undefined => {
    return flattenNodes(nodes).find(n => n.children.some(c => c.instance_id === instanceId))
  }, [])

  const getDescendants = useCallback((instanceId: string, nodes: TreeNodeResponse[]): TreeNodeResponse[] => {
    const node = flattenNodes(nodes).find(n => n.instance_id === instanceId)
    if (!node) return []
    return flattenNodes(node.children || [])
  }, [])

  const toggleNode = useCallback((nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev)
      if (next.has(nodeId)) {
        next.delete(nodeId)
      } else {
        next.add(nodeId)
      }
      return next
    })
  }, [])

  const expandNode = useCallback((nodeId: string) => {
    setExpandedNodes(prev => {
      if (prev.has(nodeId)) return prev
      return new Set([...prev, nodeId])
    })
  }, [])

  const clearSelection = useCallback(() => {
    setSelectedNodeId(null)
    setSelectedTypeName(null)
    setSelectedInstance(null)
    setForwardRefs([])
    setReverseRefs([])
  }, [])

  const navigateToTreeNode = useCallback(async (instanceId: string) => {
    const currentTree = await loadTree()
    const findAndSelect = (nodes: TreeNodeResponse[]): boolean => {
      for (const n of nodes) {
        if (n.instance_id === instanceId) {
          selectTreeNode(n)
          return true
        }
        if (n.children && findAndSelect(n.children)) {
          setExpandedNodes(prev => new Set([...prev, n.instance_id]))
          return true
        }
      }
      return false
    }
    findAndSelect(currentTree)
  }, [loadTree, selectTreeNode])

  const instanceNames = useMemo(() => {
    const map: Record<string, string> = {}
    for (const n of flattenNodes(tree)) {
      map[n.instance_id] = n.instance_name
    }
    return map
  }, [tree])

  return {
    tree,
    instanceNames,
    treeLoading,
    expandedNodes,
    selectedNodeId,
    selectedTypeName,
    selectedInstance,
    detailLoading,
    forwardRefs,
    reverseRefs,
    refsLoading,
    loadTree,
    selectTreeNode,
    selectNodeById,
    findParentNode,
    getDescendants,
    toggleNode,
    expandNode,
    clearSelection,
    navigateToTreeNode,
  }
}
