import { useState, useCallback, useRef } from 'react'
import { api } from '../api/client'
import type { CatalogVersionPin, EntityType, EntityTypeVersion, MigrationReport } from '../types'

interface UsePinManagementOptions {
  catalogVersionId: string | undefined
  loadPins: () => void
  onError: (msg: string) => void
}

export function usePinManagement({ catalogVersionId, loadPins, onError }: UsePinManagementOptions) {
  // Add pin modal state
  const [addPinOpen, setAddPinOpen] = useState(false)
  const [addPinError, setAddPinError] = useState<string | null>(null)
  const [entityTypes, setEntityTypes] = useState<EntityType[]>([])
  const [entityTypeVersions, setEntityTypeVersions] = useState<EntityTypeVersion[]>([])
  const [selectedEtvId, setSelectedEtvId] = useState('')
  const [selectedEtId, setSelectedEtId] = useState('')

  // Inline version change state (per-pin)
  const [pinVersionSelectOpen, setPinVersionSelectOpen] = useState<string | null>(null)
  const [pinVersionOptions, setPinVersionOptions] = useState<Record<string, EntityTypeVersion[]>>({})
  // Ref mirror of pinVersionOptions state — allows loadPinVersionOptions callback
  // to read the current cache without listing pinVersionOptions in its useCallback
  // dependencies. Without this, every cache update recreates the callback, causing
  // cascade recreations of handleTogglePinVersionSelect. This is a standard React
  // pattern for reading current state in stable callbacks.
  const pinVersionOptionsRef = useRef(pinVersionOptions)
  pinVersionOptionsRef.current = pinVersionOptions

  const handleOpenAddPin = useCallback(async () => {
    setAddPinError(null)
    setSelectedEtvId('')
    setSelectedEtId('')
    setEntityTypeVersions([])
    try {
      const res = await api.entityTypes.list()
      setEntityTypes(res.items || [])
    } catch { /* ignore */ }
    setAddPinOpen(true)
  }, [])

  const handleCloseAddPin = useCallback(() => {
    setAddPinOpen(false)
    setAddPinError(null)
  }, [])

  const handleSelectEntityType = useCallback(async (etId: string) => {
    setSelectedEtId(etId)
    setSelectedEtvId('')
    try {
      const res = await api.versions.list(etId)
      setEntityTypeVersions(res.items || [])
    } catch {
      setEntityTypeVersions([])
    }
  }, [])

  const handleAddPin = useCallback(async () => {
    if (!catalogVersionId || !selectedEtvId) return
    setAddPinError(null)
    try {
      await api.catalogVersions.addPin(catalogVersionId, selectedEtvId)
      setAddPinOpen(false)
      loadPins()
    } catch (e) {
      setAddPinError(e instanceof Error ? e.message : 'Failed to add pin')
    }
  }, [catalogVersionId, selectedEtvId, loadPins])

  const handleRemovePin = useCallback(async (pinId: string) => {
    if (!catalogVersionId) return
    try {
      await api.catalogVersions.removePin(catalogVersionId, pinId)
      loadPins()
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Failed to remove pin')
    }
  }, [catalogVersionId, loadPins, onError])

  // TD-75: Split into pure UI toggle + separate data loading
  const loadPinVersionOptions = useCallback(async (entityTypeId: string) => {
    if (pinVersionOptionsRef.current[entityTypeId]) return // already cached
    try {
      const res = await api.versions.list(entityTypeId)
      setPinVersionOptions(prev => ({ ...prev, [entityTypeId]: res.items || [] }))
    } catch {
      setPinVersionOptions(prev => ({ ...prev, [entityTypeId]: [] }))
    }
  }, [])

  const handleTogglePinVersionSelect = useCallback((pin: CatalogVersionPin) => {
    if (pinVersionSelectOpen === pin.pin_id) {
      setPinVersionSelectOpen(null)
      return
    }
    // Trigger version loading (separate concern from UI toggle - TD-75)
    loadPinVersionOptions(pin.entity_type_id)
    setPinVersionSelectOpen(pin.pin_id)
  }, [pinVersionSelectOpen, loadPinVersionOptions])

  const closePinVersionSelect = useCallback(() => {
    setPinVersionSelectOpen(null)
  }, [])

  // Migration preview state
  const [migrationPreview, setMigrationPreview] = useState<MigrationReport | null>(null)
  const [migrationPendingPin, setMigrationPendingPin] = useState<CatalogVersionPin | null>(null)
  const [migrationPendingEtvId, setMigrationPendingEtvId] = useState('')

  const handleUpdatePinVersion = useCallback(async (pin: CatalogVersionPin, newEtvId: string) => {
    if (!catalogVersionId) return
    setPinVersionSelectOpen(null)
    try {
      // Dry-run first to preview migration impact
      const result = await api.catalogVersions.updatePinDryRun(catalogVersionId, pin.pin_id, newEtvId)
      const hasStructuralChanges = result.migration &&
        result.migration.affected_instances > 0 && (
        result.migration.warnings.length > 0 ||
        result.migration.attribute_mappings.some(m => m.action !== 'remap')
      )
      if (hasStructuralChanges) {
        // Show preview modal
        setMigrationPreview(result.migration!)
        setMigrationPendingPin(pin)
        setMigrationPendingEtvId(newEtvId)
      } else {
        // No migration impact — apply directly
        await api.catalogVersions.updatePin(catalogVersionId, pin.pin_id, newEtvId)
        loadPins()
      }
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Failed to update pin version')
    }
  }, [catalogVersionId, loadPins, onError])

  const handleConfirmMigration = useCallback(async () => {
    if (!catalogVersionId || !migrationPendingPin) return
    try {
      await api.catalogVersions.updatePin(catalogVersionId, migrationPendingPin.pin_id, migrationPendingEtvId)
      setMigrationPreview(null)
      setMigrationPendingPin(null)
      setMigrationPendingEtvId('')
      loadPins()
    } catch (e) {
      onError(e instanceof Error ? e.message : 'Failed to apply pin version change')
    }
  }, [catalogVersionId, migrationPendingPin, migrationPendingEtvId, loadPins, onError])

  const handleCancelMigration = useCallback(() => {
    setMigrationPreview(null)
    setMigrationPendingPin(null)
    setMigrationPendingEtvId('')
  }, [])

  return {
    // Add pin modal
    addPinOpen,
    addPinError,
    entityTypes,
    entityTypeVersions,
    selectedEtvId,
    setSelectedEtvId,
    selectedEtId,
    handleOpenAddPin,
    handleCloseAddPin,
    handleSelectEntityType,
    handleAddPin,

    // Inline version change
    pinVersionSelectOpen,
    pinVersionOptions,
    handleTogglePinVersionSelect,
    closePinVersionSelect,
    handleUpdatePinVersion,

    // Remove pin
    handleRemovePin,

    // Migration preview
    migrationPreview,
    migrationPendingPin,
    handleConfirmMigration,
    handleCancelMigration,
  }
}
