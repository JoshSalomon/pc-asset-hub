import { useState, useEffect, useCallback } from 'react'
import { api } from '../api/client'
import type { EntityType, EntityTypeVersion, Attribute, Association, Enum } from '../types'

export function useEntityTypeData(entityTypeId: string | undefined, initialTab?: string) {
  const [entityType, setEntityType] = useState<EntityType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string | number>(initialTab || 'overview')

  // Attributes state
  const [attributes, setAttributes] = useState<Attribute[]>([])
  const [attrsLoading, setAttrsLoading] = useState(false)

  // Associations state
  const [associations, setAssociations] = useState<Association[]>([])
  const [assocsLoading, setAssocsLoading] = useState(false)
  const [entityTypes, setEntityTypes] = useState<EntityType[]>([])

  // Versions state
  const [versions, setVersions] = useState<EntityTypeVersion[]>([])
  const [versionsLoading, setVersionsLoading] = useState(false)

  // Enums for attribute creation
  const [enums, setEnums] = useState<Enum[]>([])

  const loadEntityType = useCallback(async () => {
    if (!entityTypeId) return
    setLoading(true)
    setError(null)
    try {
      const et = await api.entityTypes.get(entityTypeId)
      setEntityType(et)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [entityTypeId])

  const loadAttributes = useCallback(async () => {
    if (!entityTypeId) return
    setAttrsLoading(true)
    try {
      const res = await api.attributes.list(entityTypeId)
      setAttributes(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load attributes')
    } finally {
      setAttrsLoading(false)
    }
  }, [entityTypeId])

  const loadAssociations = useCallback(async () => {
    if (!entityTypeId) return
    setAssocsLoading(true)
    try {
      const res = await api.associations.list(entityTypeId)
      setAssociations(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load associations')
    } finally {
      setAssocsLoading(false)
    }
  }, [entityTypeId])

  const loadVersions = useCallback(async () => {
    if (!entityTypeId) return
    setVersionsLoading(true)
    try {
      const res = await api.versions.list(entityTypeId)
      setVersions(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load versions')
    } finally {
      setVersionsLoading(false)
    }
  }, [entityTypeId])

  useEffect(() => {
    loadEntityType()
  }, [loadEntityType])

  useEffect(() => {
    if (activeTab === 'attributes') {
      loadAttributes()
      api.enums.list().then((r) => setEnums(r.items || [])).catch(() => {})
    }
    if (activeTab === 'associations') {
      loadAssociations()
      api.entityTypes.list().then((r) => setEntityTypes(r.items || [])).catch(() => {})
    }
    if (activeTab === 'versions') loadVersions()
  }, [activeTab, loadAttributes, loadAssociations, loadVersions])

  return {
    entityType,
    loading,
    error,
    setError,
    activeTab,
    setActiveTab,
    attributes,
    setAttributes,
    attrsLoading,
    associations,
    assocsLoading,
    versions,
    versionsLoading,
    enums,
    setEnums,
    entityTypes,
    setEntityTypes,
    loadEntityType,
    loadAttributes,
    loadAssociations,
    loadVersions,
  }
}
