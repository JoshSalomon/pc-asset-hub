import { useState, useEffect, useCallback } from 'react'
import {
  Button, Modal, ModalBody, ModalHeader, ModalFooter, ModalVariant,
  Alert, AlertVariant,
  FormGroup, TextInput,
  Label,
} from '@patternfly/react-core'
import type { ExportBinding } from '../types'
import { api } from '../api/client'

interface ExporterInfo {
  name: string
  description: string
  parameter_schema: Array<{ name: string; type: string; description: string; required: boolean; default?: string }>
}

interface Props {
  catalogName: string
  catalogVersionId: string
  isAdmin: boolean
  isRW: boolean
}

export default function ExportBindingsPanel({ catalogName, catalogVersionId, isAdmin, isRW }: Props) {
  const [bindings, setBindings] = useState<ExportBinding[]>([])
  const [exporters, setExporters] = useState<ExporterInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editTarget, setEditTarget] = useState<ExportBinding | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ExportBinding | null>(null)
  const [runningId, setRunningId] = useState<string | null>(null)
  const [vsPickerBinding, setVsPickerBinding] = useState<ExportBinding | null>(null)

  const loadBindings = useCallback(async () => {
    try {
      setLoading(true)
      const [bindingsRes, exportersRes] = await Promise.all([
        api.exportBindings.list(catalogName),
        api.exporters.list(),
      ])
      setBindings(bindingsRes.items || [])
      setExporters(exportersRes.items || [])
      setError(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load bindings')
    } finally {
      setLoading(false)
    }
  }, [catalogName])

  useEffect(() => { loadBindings() }, [loadBindings])

  const handleExportNow = (binding: ExportBinding) => {
    if (binding.parameters?.virtual_server_type) {
      setVsPickerBinding(binding)
    } else {
      handleRun(binding.id)
    }
  }

  const handleRun = async (bindingId: string, vsInstanceName?: string) => {
    try {
      setRunningId(bindingId)
      setError(null)
      await api.exportBindings.run(catalogName, bindingId, vsInstanceName)
      await loadBindings()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Export failed')
    } finally {
      setRunningId(null)
    }
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return
    try {
      setError(null)
      await api.exportBindings.delete(catalogName, deleteTarget.id)
      setDeleteTarget(null)
      await loadBindings()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed')
    }
  }

  const handleToggleEnabled = async (binding: ExportBinding) => {
    try {
      setError(null)
      await api.exportBindings.update(catalogName, binding.id, { enabled: !binding.enabled })
      await loadBindings()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Update failed')
    }
  }

  if (loading) return <p>Loading export bindings...</p>

  return (
    <div style={{ padding: '1rem' }}>
      {error && <Alert variant={AlertVariant.danger} title={error} isInline style={{ marginBottom: '1rem' }} />}

      {isAdmin && (
        <Button variant="primary" onClick={() => setShowAddModal(true)} style={{ marginBottom: '1rem' }}>
          Add Export Binding
        </Button>
      )}

      {bindings.length === 0 ? (
        <p>No export bindings configured.</p>
      ) : (
        <table className="pf-v6-c-table pf-m-grid-md" role="grid" aria-label="Export bindings">
          <thead>
            <tr role="row">
              <th role="columnheader">Exporter</th>
              <th role="columnheader">Parameters</th>
              <th role="columnheader">Status</th>
              <th role="columnheader">Last Run</th>
              <th role="columnheader">Actions</th>
            </tr>
          </thead>
          <tbody>
            {bindings.map(b => (
              <tr key={b.id} role="row">
                <td role="gridcell">{b.exporter_name}</td>
                <td role="gridcell">
                  {Object.entries(b.parameters).map(([k, v]) => (
                    <Label key={k} style={{ marginRight: '0.25rem' }}>{k}={v}</Label>
                  ))}
                </td>
                <td role="gridcell">
                  {isAdmin ? (
                    <Button variant="link" onClick={() => handleToggleEnabled(b)}>
                      {b.enabled ? 'Enabled' : 'Disabled'}
                    </Button>
                  ) : (
                    b.enabled ? 'Enabled' : 'Disabled'
                  )}
                </td>
                <td role="gridcell">
                  <Label color={b.last_run_status === 'success' ? 'green' : b.last_run_status === 'failed' ? 'red' : 'grey'}>
                    {b.last_run_status}
                  </Label>
                  {b.last_run_at && <span style={{ marginLeft: '0.5rem', fontSize: '0.85em', color: '#6a6e73' }}>{new Date(b.last_run_at).toLocaleString()}</span>}
                  {b.last_run_error && <span title={b.last_run_error}> !</span>}
                </td>
                <td role="gridcell">
                  {isRW && (
                    <Button
                      variant="secondary"
                      size="sm"
                      isLoading={runningId === b.id}
                      isDisabled={runningId !== null || !b.enabled}
                      onClick={() => handleExportNow(b)}
                    >
                      Export Now
                    </Button>
                  )}
                  {isAdmin && (
                    <>
                      <Button variant="link" onClick={() => setEditTarget(b)} style={{ marginLeft: '0.5rem' }}>
                        Edit
                      </Button>
                      <Button variant="link" isDanger onClick={() => setDeleteTarget(b)} style={{ marginLeft: '0.5rem' }}>
                        Delete
                      </Button>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {showAddModal && (
        <BindingModal
          mode="create"
          catalogName={catalogName}
          catalogVersionId={catalogVersionId}
          exporters={exporters}
          onClose={() => setShowAddModal(false)}
          onSuccess={() => { setShowAddModal(false); loadBindings() }}
        />
      )}

      {editTarget && (
        <BindingModal
          mode="edit"
          catalogName={catalogName}
          catalogVersionId={catalogVersionId}
          exporters={exporters}
          binding={editTarget}
          onClose={() => setEditTarget(null)}
          onSuccess={() => { setEditTarget(null); loadBindings() }}
        />
      )}

      {vsPickerBinding && (
        <VSInstancePickerModal
          catalogName={catalogName}
          vsTypeName={vsPickerBinding.parameters?.virtual_server_type || ''}
          onClose={() => setVsPickerBinding(null)}
          onSelect={(instanceName) => {
            setVsPickerBinding(null)
            handleRun(vsPickerBinding.id, instanceName)
          }}
        />
      )}

      {deleteTarget && (
        <Modal variant={ModalVariant.small} isOpen onClose={() => setDeleteTarget(null)}>
          <ModalHeader title="Delete Export Binding" />
          <ModalBody>
            Are you sure you want to delete the <strong>{deleteTarget.exporter_name}</strong> export binding?
          </ModalBody>
          <ModalFooter>
            <Button variant="danger" onClick={handleDeleteConfirm}>Delete</Button>
            <Button variant="link" onClick={() => setDeleteTarget(null)}>Cancel</Button>
          </ModalFooter>
        </Modal>
      )}
    </div>
  )
}

function BindingModal({ mode, catalogName, catalogVersionId, exporters, binding, onClose, onSuccess }: {
  mode: 'create' | 'edit'
  catalogName: string
  catalogVersionId: string
  exporters: ExporterInfo[]
  binding?: ExportBinding
  onClose: () => void
  onSuccess: () => void
}) {
  const [selectedExporter, setSelectedExporter] = useState(binding?.exporter_name || '')
  const [params, setParams] = useState<Record<string, string>>(binding?.parameters ? { ...binding.parameters } : {})
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [entityTypeNames, setEntityTypeNames] = useState<string[]>([])
  const [pinLoadError, setPinLoadError] = useState<string | null>(null)

  useEffect(() => {
    if (catalogVersionId) {
      api.catalogVersions.listPins(catalogVersionId)
        .then(res => setEntityTypeNames((res.items || []).map((p: { entity_type_name: string }) => p.entity_type_name).filter(Boolean).sort()))
        .catch(e => setPinLoadError(e instanceof Error ? e.message : 'Failed to load entity types'))
    }
  }, [catalogVersionId])

  const exporter = exporters.find(e => e.name === selectedExporter)

  useEffect(() => {
    if (exporter && mode === 'create') {
      const defaults: Record<string, string> = {}
      for (const p of exporter.parameter_schema) {
        if (p.default) defaults[p.name] = p.default
      }
      setParams(defaults)
    }
  }, [exporter, mode])

  const hasEmptyRequiredParams = exporter?.parameter_schema.some(
    p => p.required && !params[p.name]
  ) ?? false

  const handleSubmit = async () => {
    try {
      setSubmitting(true)
      setError(null)
      if (mode === 'create') {
        await api.exportBindings.create(catalogName, {
          exporter_name: selectedExporter,
          parameters: params,
        })
      } else if (binding) {
        await api.exportBindings.update(catalogName, binding.id, { parameters: params })
      }
      onSuccess()
    } catch (e) {
      setError(e instanceof Error ? e.message : `Failed to ${mode} binding`)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal variant={ModalVariant.medium} isOpen onClose={onClose}>
      <ModalHeader title={mode === 'create' ? 'Add Export Binding' : 'Edit Export Binding'} />
      <ModalBody>
        {error && <Alert variant={AlertVariant.danger} title={error} isInline style={{ marginBottom: '1rem' }} />}
        {pinLoadError && <Alert variant={AlertVariant.warning} title={`Failed to load entity types: ${pinLoadError}`} isInline style={{ marginBottom: '1rem' }} />}
        {mode === 'create' && (
          <FormGroup label="Exporter" isRequired fieldId="exporter-select">
            <select
              id="exporter-select"
              value={selectedExporter}
              onChange={e => setSelectedExporter(e.target.value)}
              style={{ width: '100%', padding: '6px 12px' }}
              aria-label="Select exporter"
            >
              <option value="">Select an exporter...</option>
              {exporters.map(ex => (
                <option key={ex.name} value={ex.name}>{ex.name} — {ex.description}</option>
              ))}
            </select>
          </FormGroup>
        )}
        {mode === 'edit' && (
          <FormGroup label="Exporter" fieldId="exporter-display">
            <p>{selectedExporter}</p>
          </FormGroup>
        )}
        {exporter && exporter.parameter_schema.map(p => (
          <FormGroup key={p.name} label={`${p.name}${p.required ? ' *' : ''}`} fieldId={`param-${p.name}`}>
            {p.type === 'entity_type' ? (
              <select
                id={`param-${p.name}`}
                aria-label={p.name}
                value={params[p.name] || ''}
                onChange={e => setParams(prev => ({ ...prev, [p.name]: e.target.value }))}
                style={{ width: '100%', padding: '6px 12px' }}
              >
                <option value="">Select entity type...</option>
                {entityTypeNames.map(name => (
                  <option key={name} value={name}>{name}</option>
                ))}
              </select>
            ) : (
              <TextInput
                id={`param-${p.name}`}
                aria-label={p.name}
                value={params[p.name] || ''}
                onChange={(_e, v) => setParams(prev => ({ ...prev, [p.name]: v }))}
                placeholder={p.description}
              />
            )}
          </FormGroup>
        ))}
      </ModalBody>
      <ModalFooter>
        <Button
          variant="primary"
          onClick={handleSubmit}
          isDisabled={!selectedExporter || submitting || hasEmptyRequiredParams}
          isLoading={submitting}
        >
          {mode === 'create' ? 'Create' : 'Save'}
        </Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}

function VSInstancePickerModal({ catalogName, vsTypeName, onClose, onSelect }: {
  catalogName: string
  vsTypeName: string
  onClose: () => void
  onSelect: (instanceName: string) => void
}) {
  const [instances, setInstances] = useState<Array<{ id: string; name: string }>>([])
  const [selected, setSelected] = useState('')
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)

  useEffect(() => {
    api.instances.list(catalogName, vsTypeName, { limit: 100 })
      .then(res => setInstances((res.items || []).map((i: { id: string; name: string }) => ({ id: i.id, name: i.name }))))
      .catch(e => setLoadError(e instanceof Error ? e.message : 'Failed to load instances'))
      .finally(() => setLoading(false))
  }, [catalogName, vsTypeName])

  return (
    <Modal variant={ModalVariant.small} isOpen onClose={onClose}>
      <ModalHeader title="Select Virtual Server" />
      <ModalBody>
        {loading ? (
          <p>Loading instances...</p>
        ) : loadError ? (
          <Alert variant={AlertVariant.danger} title={loadError} isInline />
        ) : instances.length === 0 ? (
          <p>No {vsTypeName} instances found in this catalog.</p>
        ) : (
          <FormGroup label="Virtual Server Instance" isRequired fieldId="vs-instance-select">
            <select
              id="vs-instance-select"
              aria-label="Select virtual server instance"
              value={selected}
              onChange={e => setSelected(e.target.value)}
              style={{ width: '100%', padding: '6px 12px' }}
            >
              <option value="">Select an instance...</option>
              {instances.map(inst => (
                <option key={inst.id} value={inst.name}>{inst.name}</option>
              ))}
            </select>
          </FormGroup>
        )}
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => onSelect(selected)} isDisabled={!selected}>
          Export
        </Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
