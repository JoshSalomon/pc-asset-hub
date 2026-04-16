import { useState, useCallback } from 'react'
import { api, setAuthRole } from '../api/client'
import type { EntityInstance, SnapshotAttribute, Role } from '../types'
import { buildTypedAttrs } from '../utils/buildTypedAttrs'

export function useInstances(catalogName: string | undefined, entityTypeName: string, schemaAttrs: SnapshotAttribute[], role: Role) {
  const [instances, setInstances] = useState<EntityInstance[]>([])
  const [instTotal, setInstTotal] = useState(0)
  const [instLoading, setInstLoading] = useState(false)

  // Create instance modal
  const [createOpen, setCreateOpen] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  // Edit instance modal
  const [editTarget, setEditTarget] = useState<EntityInstance | null>(null)
  const [editError, setEditError] = useState<string | null>(null)

  // Delete instance modal
  const [deleteTarget, setDeleteTarget] = useState<EntityInstance | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const loadInstances = useCallback(async () => {
    if (!catalogName || !entityTypeName) return
    setAuthRole(role)
    setInstLoading(true)
    try {
      const res = await api.instances.list(catalogName, entityTypeName)
      setInstances(res.items || [])
      setInstTotal(res.total)
    } catch {
      setInstances([])
      setInstTotal(0)
    } finally {
      setInstLoading(false)
    }
  }, [catalogName, entityTypeName, role])

  const handleCreate = async (name: string, description: string, rawAttrs: Record<string, string>) => {
    if (!catalogName || !entityTypeName || !name.trim()) return
    setCreateError(null)
    try {
      const attrs = buildTypedAttrs(rawAttrs, schemaAttrs)
      await api.instances.create(catalogName, entityTypeName, {
        name: name.trim(),
        description: description.trim() || undefined,
        attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
      })
      setCreateOpen(false)
      await loadInstances()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const openCreate = () => {
    setCreateOpen(true)
  }

  const closeCreate = () => {
    setCreateOpen(false)
    setCreateError(null)
  }

  const openEdit = (inst: EntityInstance) => {
    setEditTarget(inst)
    setEditError(null)
  }

  const closeEdit = () => {
    setEditTarget(null)
    setEditError(null)
  }

  const handleEdit = async (version: number, name: string, description: string, rawAttrs: Record<string, string>) => {
    if (!catalogName || !entityTypeName || !editTarget) return
    setEditError(null)
    try {
      const attrs = buildTypedAttrs(rawAttrs, schemaAttrs, true)
      await api.instances.update(catalogName, entityTypeName, editTarget.id, {
        version,
        name: name !== editTarget.name ? name : undefined,
        description: description !== editTarget.description ? description : undefined,
        attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
      })
      setEditTarget(null)
      await loadInstances()
    } catch (e) {
      setEditError(e instanceof Error ? e.message : 'Failed to update')
    }
  }

  const openDelete = (inst: EntityInstance) => {
    setDeleteTarget(inst)
    setDeleteError(null)
  }

  const closeDelete = () => {
    setDeleteTarget(null)
    setDeleteError(null)
  }

  const handleDelete = async () => {
    if (!catalogName || !entityTypeName || !deleteTarget) return
    setDeleteError(null)
    try {
      await api.instances.delete(catalogName, entityTypeName, deleteTarget.id)
      setDeleteTarget(null)
      await loadInstances()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  return {
    instances, instTotal, instLoading,
    createOpen, openCreate, closeCreate,
    createError, handleCreate,
    editTarget, openEdit, closeEdit,
    editError, handleEdit,
    deleteTarget, openDelete, closeDelete,
    deleteError, handleDelete,
    loadInstances,
  }
}
