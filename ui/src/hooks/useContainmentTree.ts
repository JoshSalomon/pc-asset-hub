import { useState, useCallback } from 'react'
import { api } from '../api/client'
import type { TreeNodeResponse, EntityInstance, ReferenceDetail } from '../types'

export function useContainmentTree(catalogName: string | undefined) {
  const [tree, setTree] = useState<TreeNodeResponse[]>([])
  const [treeLoading, setTreeLoading] = useState(false)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)

  const [selectedInstance, setSelectedInstance] = useState<EntityInstance | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [forwardRefs, setForwardRefs] = useState<ReferenceDetail[]>([])
  const [reverseRefs, setReverseRefs] = useState<ReferenceDetail[]>([])
  const [refsLoading, setRefsLoading] = useState(false)

  const loadTree = useCallback(async () => {
    if (!catalogName) return
    setTreeLoading(true)
    try {
      const t = await api.instances.tree(catalogName)
      setTree(t || [])
    } catch {
      setTree([])
    } finally {
      setTreeLoading(false)
    }
  }, [catalogName])

  const selectTreeNode = useCallback(async (node: TreeNodeResponse) => {
    setSelectedNodeId(node.instance_id)
    setDetailLoading(true)
    try {
      const inst = await api.instances.get(catalogName!, node.entity_type_name, node.instance_id)
      setSelectedInstance(inst)
    } catch {
      setSelectedInstance(null)
    } finally {
      setDetailLoading(false)
    }

    // Load references
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

  const navigateToTreeNode = useCallback((instanceId: string) => {
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
    findAndSelect(tree)
  }, [tree, selectTreeNode])

  return {
    tree,
    treeLoading,
    expandedNodes,
    selectedNodeId,
    selectedInstance,
    detailLoading,
    forwardRefs,
    reverseRefs,
    refsLoading,
    loadTree,
    selectTreeNode,
    toggleNode,
    expandNode,
    navigateToTreeNode,
  }
}
