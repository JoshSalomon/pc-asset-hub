import { useState, useCallback } from 'react'
import { api } from '../api/client'

interface UseInlineEditOptions {
  catalogVersionId: string | undefined
  onSuccess: () => void
  onError: (msg: string) => void
}

export function useInlineEdit({ catalogVersionId, onSuccess, onError }: UseInlineEditOptions) {
  const [editingLabel, setEditingLabel] = useState(false)
  const [editLabelValue, setEditLabelValue] = useState('')

  const [editingDesc, setEditingDesc] = useState(false)
  const [editDescValue, setEditDescValue] = useState('')

  const startEditLabel = useCallback((currentValue: string) => {
    setEditLabelValue(currentValue)
    setEditingLabel(true)
  }, [])

  const cancelEditLabel = useCallback(() => {
    setEditingLabel(false)
  }, [])

  const handleSaveLabel = useCallback(async () => {
    if (!catalogVersionId) return
    try {
      await api.catalogVersions.update(catalogVersionId, { version_label: editLabelValue })
      setEditingLabel(false)
      onSuccess()
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Failed to update version label')
    }
  }, [catalogVersionId, editLabelValue, onSuccess, onError])

  const startEditDesc = useCallback((currentValue: string) => {
    setEditDescValue(currentValue)
    setEditingDesc(true)
  }, [])

  const cancelEditDesc = useCallback(() => {
    setEditingDesc(false)
  }, [])

  const handleSaveDescription = useCallback(async () => {
    if (!catalogVersionId) return
    try {
      await api.catalogVersions.update(catalogVersionId, { description: editDescValue })
      setEditingDesc(false)
      onSuccess()
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Failed to update description')
    }
  }, [catalogVersionId, editDescValue, onSuccess, onError])

  return {
    editingLabel,
    editLabelValue,
    setEditLabelValue,
    startEditLabel,
    cancelEditLabel,
    handleSaveLabel,

    editingDesc,
    editDescValue,
    setEditDescValue,
    startEditDesc,
    cancelEditDesc,
    handleSaveDescription,
  }
}
