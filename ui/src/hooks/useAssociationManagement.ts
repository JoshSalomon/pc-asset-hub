import { useState, useCallback } from 'react'
import { api } from '../api/client'
import type { Association } from '../types'
import type { AddAssociationValues } from '../components/AddAssociationModal'

interface EditAssociationValues {
  name: string
  type: string
  sourceRole: string
  targetRole: string
  sourceCardinality: string
  targetCardinality: string
}

interface UseAssociationManagementOptions {
  entityTypeId: string | undefined
  onRefresh: () => void
  setError: React.Dispatch<React.SetStateAction<string | null>>
}

export function useAssociationManagement({
  entityTypeId,
  onRefresh,
  setError,
}: UseAssociationManagementOptions) {
  const [addAssocOpen, setAddAssocOpen] = useState(false)
  const [addAssocError, setAddAssocError] = useState<string | null>(null)

  const [editAssocOpen, setEditAssocOpen] = useState(false)
  const [editAssocError, setEditAssocError] = useState<string | null>(null)
  const [editAssocData, setEditAssocData] = useState({
    name: '', type: '', sourceRole: '', targetRole: '',
    sourceCardinality: '0..n', targetCardinality: '0..n',
  })

  const handleAddAssociation = useCallback(async (values: AddAssociationValues) => {
    setAddAssocError(null)
    if (values.sourceCardCustom && (!values.sourceCardMin.trim() || !values.sourceCardMax.trim())) {
      setAddAssocError('Source cardinality: both min and max are required for custom values')
      return
    }
    if (values.targetCardCustom && (!values.targetCardMin.trim() || !values.targetCardMax.trim())) {
      setAddAssocError('Target cardinality: both min and max are required for custom values')
      return
    }
    if (!entityTypeId || !values.targetId || !values.type) return
    try {
      const srcCard = values.sourceCardCustom ? `${values.sourceCardMin}..${values.sourceCardMax}` : values.sourceCardinality
      const tgtCard = values.targetCardCustom ? `${values.targetCardMin}..${values.targetCardMax}` : values.targetCardinality
      await api.associations.create(entityTypeId, {
        target_entity_type_id: values.targetId,
        type: values.type,
        name: values.name,
        source_role: values.sourceRole || undefined,
        target_role: values.targetRole || undefined,
        source_cardinality: srcCard,
        target_cardinality: tgtCard,
      })
      setAddAssocOpen(false)
      onRefresh()
    } catch (e) {
      setAddAssocError(e instanceof Error ? e.message : 'Failed to create association')
    }
  }, [entityTypeId, onRefresh])

  const handleDeleteAssociation = useCallback(async (assocName: string) => {
    if (!entityTypeId) return
    try {
      await api.associations.delete(entityTypeId, assocName)
      onRefresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete association')
    }
  }, [entityTypeId, onRefresh, setError])

  const openEditAssoc = useCallback((assoc: Association) => {
    setEditAssocData({
      name: assoc.name,
      type: assoc.type,
      sourceRole: assoc.source_role || '',
      targetRole: assoc.target_role || '',
      sourceCardinality: assoc.source_cardinality || '0..n',
      targetCardinality: assoc.target_cardinality || '0..n',
    })
    setEditAssocOpen(true)
  }, [])

  const handleEditAssociationSave = useCallback(async (data: EditAssociationValues) => {
    if (!entityTypeId) return
    setEditAssocError(null)
    try {
      const req: Record<string, string | undefined> = {}
      if (data.name !== editAssocData.name) req.name = data.name
      req.type = data.type
      req.source_role = data.sourceRole
      req.target_role = data.targetRole
      req.source_cardinality = data.sourceCardinality
      req.target_cardinality = data.targetCardinality
      await api.associations.edit(entityTypeId, editAssocData.name, req)
      setEditAssocOpen(false)
      onRefresh()
    } catch (e) {
      setEditAssocError(e instanceof Error ? e.message : 'Failed to edit association')
      throw e  // Re-throw so the modal's own try/catch can also display the error
    }
  }, [entityTypeId, editAssocData.name, onRefresh])

  return {
    addAssocOpen,
    setAddAssocOpen,
    addAssocError,
    setAddAssocError,
    handleAddAssociation,

    editAssocOpen,
    setEditAssocOpen,
    editAssocError,
    setEditAssocError,
    editAssocData,
    openEditAssoc,
    handleEditAssociationSave,

    handleDeleteAssociation,
  }
}
