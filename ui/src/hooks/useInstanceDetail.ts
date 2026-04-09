import { useState, useCallback } from 'react'
import { api, setAuthRole } from '../api/client'
import type { EntityInstance, SnapshotAssociation, ReferenceDetail, Role } from '../types'

export function useInstanceDetail(catalogName: string | undefined, entityTypeName: string, schemaAssocs: SnapshotAssociation[], role: Role) {
  const [selectedInstance, setSelectedInstance] = useState<EntityInstance | null>(null)
  const [parentName, setParentName] = useState<string>('')
  const [children, setChildren] = useState<EntityInstance[]>([])
  const [childrenLoading, setChildrenLoading] = useState(false)
  const [forwardRefs, setForwardRefs] = useState<ReferenceDetail[]>([])
  const [reverseRefs, setReverseRefs] = useState<ReferenceDetail[]>([])
  const [refsLoading, setRefsLoading] = useState(false)

  const selectInstance = useCallback(async (instanceId: string | null) => {
    setParentName('')
    if (!instanceId || !catalogName || !entityTypeName) {
      setSelectedInstance(null)
      setChildren([])
      setForwardRefs([])
      setReverseRefs([])
      return
    }
    // TD-49: Set auth role before any API calls
    setAuthRole(role)
    // TD-50: Always re-fetch instance by ID to get fresh data
    let inst: EntityInstance
    try {
      inst = await api.instances.get(catalogName, entityTypeName, instanceId)
    } catch {
      setSelectedInstance(null)
      setChildren([])
      setForwardRefs([])
      setReverseRefs([])
      return
    }
    setSelectedInstance(inst)
    // Resolve parent name if instance is contained
    if (inst.parent_instance_id) {
      try {
        const parent = await api.instances.get(catalogName, entityTypeName, inst.parent_instance_id)
        setParentName(parent.name)
      } catch { setParentName(inst.parent_instance_id) }
    }
    // Load contained children — find containment associations pointing from this entity type
    setChildrenLoading(true)
    try {
      const containmentAssocs = schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing')
      const allChildren: EntityInstance[] = []
      for (const assoc of containmentAssocs) {
        const childTypeName = assoc.target_entity_type_name
        try {
          const res = await api.instances.listContained(catalogName, entityTypeName, inst.id, childTypeName)
          allChildren.push(...(res.items || []))
        } catch { /* ignore if child type not found */ }
      }
      setChildren(allChildren)
    } catch {
      setChildren([])
    } finally {
      setChildrenLoading(false)
    }
    // Load references
    setRefsLoading(true)
    try {
      const [fwd, rev] = await Promise.all([
        api.links.forwardRefs(catalogName, entityTypeName, inst.id),
        api.links.reverseRefs(catalogName, entityTypeName, inst.id),
      ])
      setForwardRefs(fwd || [])
      setReverseRefs(rev || [])
    } catch {
      setForwardRefs([])
      setReverseRefs([])
    } finally {
      setRefsLoading(false)
    }
  }, [catalogName, entityTypeName, schemaAssocs, role])

  const clearSelection = useCallback(() => {
    setSelectedInstance(null)
    setChildren([])
    setForwardRefs([])
    setReverseRefs([])
  }, [])

  return {
    selectedInstance, selectInstance, clearSelection,
    parentName, children, childrenLoading,
    forwardRefs, reverseRefs, refsLoading,
  }
}
