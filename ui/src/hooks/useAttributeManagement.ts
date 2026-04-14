import { useState, useCallback } from 'react'
import { api } from '../api/client'
import type { Attribute, TypeDefinition } from '../types'
import type { AddAttributeValues } from '../components/AddAttributeModal'
import type { EditAttributeValues } from '../components/EditAttributeModal'

interface UseAttributeManagementOptions {
  entityTypeId: string | undefined
  attributes: Attribute[]
  typeDefinitions: TypeDefinition[]
  onRefresh: () => void
  setAttributes: React.Dispatch<React.SetStateAction<Attribute[]>>
  setTypeDefinitions: React.Dispatch<React.SetStateAction<TypeDefinition[]>>
  setError: React.Dispatch<React.SetStateAction<string | null>>
}

export function useAttributeManagement({
  entityTypeId,
  attributes,
  typeDefinitions,
  onRefresh,
  setAttributes,
  setTypeDefinitions,
  setError,
}: UseAttributeManagementOptions) {
  // Add attribute modal
  const [addAttrOpen, setAddAttrOpen] = useState(false)
  const [addAttrError, setAddAttrError] = useState<string | null>(null)

  // Edit attribute modal
  const [editAttrOpen, setEditAttrOpen] = useState(false)
  const [editAttrOrigName, setEditAttrOrigName] = useState('')
  const [editAttrError, setEditAttrError] = useState<string | null>(null)

  // Copy attributes modal
  const [copyAttrsOpen, setCopyAttrsOpen] = useState(false)
  const [copyAttrsSourceId, setCopyAttrsSourceId] = useState('')
  const [sourceAttributes, setSourceAttributes] = useState<Attribute[]>([])
  const [sourceLatestVersion, setSourceLatestVersion] = useState(1)
  const [selectedCopyAttrs, setSelectedCopyAttrs] = useState<string[]>([])
  const [copyAttrsError, setCopyAttrsError] = useState<string | null>(null)

  const handleAddAttribute = useCallback(async (values: AddAttributeValues) => {
    if (!entityTypeId || !values.name.trim() || !values.typeDefinitionVersionId) return
    setAddAttrError(null)
    try {
      await api.attributes.add(entityTypeId, {
        name: values.name.trim(),
        description: values.description.trim() || undefined,
        type_definition_version_id: values.typeDefinitionVersionId,
        required: values.required,
      })
      setAddAttrOpen(false)
      onRefresh()
    } catch (e) {
      setAddAttrError(e instanceof Error ? e.message : 'Failed to add attribute')
    }
  }, [entityTypeId, onRefresh])

  const handleRemoveAttribute = useCallback(async (name: string) => {
    if (!entityTypeId) return
    try {
      await api.attributes.remove(entityTypeId, name)
      onRefresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove attribute')
    }
  }, [entityTypeId, onRefresh, setError])

  const handleReorderAttribute = useCallback(async (index: number, direction: 'up' | 'down') => {
    if (!entityTypeId) return
    const newAttrs = [...attributes]
    const swapIndex = direction === 'up' ? index - 1 : index + 1
    if (swapIndex < 0 || swapIndex >= newAttrs.length) return
    ;[newAttrs[index], newAttrs[swapIndex]] = [newAttrs[swapIndex], newAttrs[index]]
    try {
      await api.attributes.reorder(entityTypeId, newAttrs.map((a) => a.id))
      setAttributes(newAttrs)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to reorder')
    }
  }, [entityTypeId, attributes, setAttributes, setError])

  const openEditAttr = useCallback((attr: Attribute) => {
    setEditAttrOrigName(attr.name)
    setEditAttrError(null)
    setEditAttrOpen(true)
    if (typeDefinitions.length === 0) {
      api.typeDefinitions.list().then((r) => setTypeDefinitions(r.items || [])).catch(() => {})
    }
  }, [typeDefinitions.length, setTypeDefinitions])

  const handleEditAttribute = useCallback(async (values: EditAttributeValues) => {
    if (!entityTypeId) return
    setEditAttrError(null)
    try {
      const data: Record<string, string | boolean | undefined> = {}
      if (values.name !== editAttrOrigName) data.name = values.name
      if (values.description !== undefined) data.description = values.description
      if (values.typeDefinitionVersionId) data.type_definition_version_id = values.typeDefinitionVersionId
      data.required = values.required
      await api.attributes.edit(entityTypeId, editAttrOrigName, data)
      setEditAttrOpen(false)
      onRefresh()
    } catch (e) {
      setEditAttrError(e instanceof Error ? e.message : 'Failed to edit attribute')
    }
  }, [entityTypeId, editAttrOrigName, onRefresh])

  const handleLoadSourceAttrs = useCallback(async (sourceId: string) => {
    setCopyAttrsSourceId(sourceId)
    setSelectedCopyAttrs([])
    try {
      const [attrRes, versRes, tdRes] = await Promise.all([
        api.attributes.list(sourceId),
        api.versions.list(sourceId),
        api.typeDefinitions.list(),
      ])
      setSourceAttributes(attrRes.items || [])
      setTypeDefinitions(tdRes.items || [])
      const srcVersions = versRes.items || []
      const latest = srcVersions.length > 0 ? Math.max(...srcVersions.map((v: { version: number }) => v.version)) : 1
      setSourceLatestVersion(latest)
    } catch {
      setSourceAttributes([])
    }
  }, [setTypeDefinitions])

  const handleCopyAttributes = useCallback(async (attrNames?: string[]) => {
    const attrsToUse = attrNames || selectedCopyAttrs
    if (!entityTypeId || !copyAttrsSourceId || attrsToUse.length === 0) return
    setCopyAttrsError(null)
    try {
      await api.attributes.copyFrom(entityTypeId, {
        source_entity_type_id: copyAttrsSourceId,
        source_version: sourceLatestVersion,
        attribute_names: attrsToUse,
      })
      setCopyAttrsOpen(false)
      setCopyAttrsSourceId('')
      setSourceAttributes([])
      setSelectedCopyAttrs([])
      onRefresh()
    } catch (e) {
      setCopyAttrsError(e instanceof Error ? e.message : 'Failed to copy attributes')
    }
  }, [entityTypeId, copyAttrsSourceId, selectedCopyAttrs, sourceLatestVersion, onRefresh])

  return {
    // Add attribute
    addAttrOpen,
    setAddAttrOpen,
    addAttrError,
    setAddAttrError,
    handleAddAttribute,

    // Edit attribute
    editAttrOpen,
    setEditAttrOpen,
    editAttrOrigName,
    editAttrError,
    setEditAttrError,
    openEditAttr,
    handleEditAttribute,

    // Remove / reorder
    handleRemoveAttribute,
    handleReorderAttribute,

    // Copy attributes
    copyAttrsOpen,
    setCopyAttrsOpen,
    copyAttrsSourceId,
    setCopyAttrsSourceId,
    sourceAttributes,
    sourceLatestVersion,
    selectedCopyAttrs,
    setSelectedCopyAttrs,
    copyAttrsError,
    setCopyAttrsError,
    handleLoadSourceAttrs,
    handleCopyAttributes,
  }
}
