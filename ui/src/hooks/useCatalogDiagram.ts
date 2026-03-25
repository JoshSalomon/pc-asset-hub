import { useState, useCallback, useEffect, useRef } from 'react'
import { api } from '../api/client'
import type { DiagramEntityType } from '../components/EntityTypeDiagram'

export function useCatalogDiagram(catalogVersionId?: string) {
  const [diagramData, setDiagramData] = useState<DiagramEntityType[]>([])
  const [diagramLoading, setDiagramLoading] = useState(false)
  const [diagramError, setDiagramError] = useState<string | null>(null)
  const loadedRef = useRef(false)

  // Reset when catalogVersionId changes so new data can be loaded
  useEffect(() => {
    loadedRef.current = false
    setDiagramData([])
    setDiagramError(null)
  }, [catalogVersionId])

  const loadDiagram = useCallback(async () => {
    if (!catalogVersionId || loadedRef.current) return
    setDiagramLoading(true)
    setDiagramError(null)
    try {
      const pinsRes = await api.catalogVersions.listPins(catalogVersionId)
      const pins = pinsRes.items || []
      const snapshots: DiagramEntityType[] = await Promise.all(
        pins.map(async (pin) => {
          const snap = await api.versions.snapshot(pin.entity_type_id, pin.version)
          return {
            entityType: snap.entity_type,
            version: pin.version,
            attributes: snap.attributes || [],
            associations: snap.associations || [],
          }
        })
      )
      setDiagramData(snapshots)
      loadedRef.current = true
    } catch (e) {
      setDiagramError(e instanceof Error ? e.message : 'Failed to load diagram')
    } finally {
      setDiagramLoading(false)
    }
  }, [catalogVersionId])

  return { diagramData, diagramLoading, diagramError, loadDiagram }
}
