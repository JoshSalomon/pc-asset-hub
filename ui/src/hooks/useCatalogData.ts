import { useState, useEffect, useCallback } from 'react'
import { api, setAuthRole } from '../api/client'
import type { Catalog, CatalogVersionPin, SnapshotAttribute, SnapshotAssociation, Role } from '../types'

export function useCatalogData(catalogName: string | undefined, role: Role) {
  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pins, setPins] = useState<CatalogVersionPin[]>([])
  const [activeTab, setActiveTab] = useState<string>('')

  // Schema attributes for the active entity type
  const [schemaAttrs, setSchemaAttrs] = useState<SnapshotAttribute[]>([])
  const [schemaAssocs, setSchemaAssocs] = useState<SnapshotAssociation[]>([])

  // Enum values cache for enum dropdowns (keyed by type_definition_version_id)
  const [enumValues, setEnumValues] = useState<Record<string, string[]>>({})

  const loadCatalog = useCallback(async () => {
    if (!catalogName) return
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const cat = await api.catalogs.get(catalogName)
      setCatalog(cat)
      // Load pins from the CV
      const pinsRes = await api.catalogVersions.listPins(cat.catalog_version_id)
      setPins(pinsRes.items || [])
      if (pinsRes.items?.length) {
        setActiveTab(prev => prev || pinsRes.items![0].entity_type_name)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalog')
    } finally {
      setLoading(false)
    }
  }, [catalogName, role])

  useEffect(() => { loadCatalog() }, [loadCatalog])

  const loadSchema = useCallback(async () => {
    if (!activeTab || !pins.length) return
    const pin = pins.find(p => p.entity_type_name === activeTab)
    if (!pin) return
    try {
      const snapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
      setSchemaAttrs(snapshot.attributes || [])
      setSchemaAssocs(snapshot.associations || [])
      // For enum-type attributes, extract values from constraints
      const enumCache: Record<string, string[]> = {}
      for (const attr of snapshot.attributes || []) {
        if (attr.base_type === 'enum' && attr.type_definition_version_id) {
          const constraintValues = (attr.constraints?.values as string[]) || []
          if (constraintValues.length > 0) {
            enumCache[attr.type_definition_version_id] = constraintValues
          }
        }
      }
      setEnumValues(enumCache)
    } catch { /* ignore */ }
  }, [activeTab, pins])

  useEffect(() => { loadSchema() }, [loadSchema])

  return {
    catalog, loading, error, setError,
    pins, activeTab, setActiveTab,
    schemaAttrs, schemaAssocs, enumValues,
    loadCatalog,
  }
}
