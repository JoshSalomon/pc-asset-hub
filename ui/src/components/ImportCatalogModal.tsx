import { useState, useRef, useEffect, useCallback } from 'react'
import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Checkbox,
  Form,
  FormGroup,
  TextInput,
  Alert,
  Spinner,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api } from '../api/client'
import { DNS_LABEL_RE } from '../utils/dnsLabel'

interface DryRunCollision {
  type: string
  name: string
  resolution: string
  version?: number
  detail: string
}

interface DryRunResponse {
  status: string
  collisions: DryRunCollision[]
  summary: { total_entities: number; conflicts: number; identical: number; new: number }
}

interface ImportResponse {
  status: string
  catalog_name: string
  catalog_id: string
  types_created: number
  types_reused: number
  instances_created: number
  links_created: number
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSuccess: (catalogName: string) => void
}

type Step = 'upload' | 'collisions' | 'confirm' | 'done'

export default function ImportCatalogModal({ isOpen, onClose, onSuccess }: Props) {
  const [step, setStep] = useState<Step>('upload')
  const [fileData, setFileData] = useState<unknown>(null)
  const [fileName, setFileName] = useState('')
  const [catalogName, setCatalogName] = useState('')
  const [cvLabel, setCvLabel] = useState('')
  const [prefix, setPrefix] = useState('')
  const [suffix, setSuffix] = useState('')
  const [nameError, setNameError] = useState<string | null>(null)
  const [cvLabelError, setCvLabelError] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [dryRunResult, setDryRunResult] = useState<DryRunResponse | null>(null)
  const [importResult, setImportResult] = useState<ImportResponse | null>(null)
  const [reuseExisting, setReuseExisting] = useState<string[]>([])
  const [dragging, setDragging] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const dragCounterRef = useRef(0)

  // Document-level handlers prevent the browser from opening files that
  // miss the dropzone, and ensure Chromium sees a valid drop target.
  useEffect(() => {
    if (!isOpen) return
    const prevent = (e: DragEvent) => e.preventDefault()
    document.addEventListener('dragenter', prevent)
    document.addEventListener('dragover', prevent)
    document.addEventListener('drop', prevent)
    return () => {
      document.removeEventListener('dragenter', prevent)
      document.removeEventListener('dragover', prevent)
      document.removeEventListener('drop', prevent)
    }
  }, [isOpen])

  const reset = () => {
    setStep('upload')
    setFileData(null)
    setFileName('')
    setCatalogName('')
    setCvLabel('')
    setPrefix('')
    setSuffix('')
    setNameError(null)
    setCvLabelError(null)
    setError(null)
    setLoading(false)
    setDryRunResult(null)
    setImportResult(null)
    setReuseExisting([])
    setDragging(false)
    dragCounterRef.current = 0
  }

  const handleClose = () => {
    reset()
    onClose()
  }

  const processFile = useCallback((file: File) => {
    if (file.size > 50 * 1024 * 1024) {
      setError('File too large (max 50 MB)')
      setFileData(null)
      return
    }
    setFileName(file.name)
    const reader = new FileReader()
    reader.onload = (ev) => {
      try {
        const parsed = JSON.parse(ev.target?.result as string)
        if (!parsed.format_version || !parsed.catalog || !parsed.catalog_version || !parsed.entity_types) {
          setError('Not a valid catalog export file — missing required fields (format_version, catalog, catalog_version, entity_types)')
          setFileData(null)
          return
        }
        setFileData(parsed)
        setCatalogName(parsed.catalog?.name || '')
        setCvLabel(parsed.catalog_version?.label || '')
        setError(null)
      } catch {
        setError('Invalid JSON file')
        setFileData(null)
      }
    }
    reader.readAsText(file)
  }, [])

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    processFile(file)
  }

  // Native event listeners bypass React's portal event delegation.
  // PatternFly Modal renders via createPortal — React synthetic drag events
  // may not call preventDefault() on the native event in time for Chromium.
  // Uses a drag counter to handle dragLeave on child elements correctly.
  const dropzoneCallbackRef = useCallback((el: HTMLDivElement | null) => {
    if (!el) {
      dragCounterRef.current = 0
      return
    }
    el.addEventListener('dragenter', (e: DragEvent) => {
      e.preventDefault()
      dragCounterRef.current++
      setDragging(true)
    })
    el.addEventListener('dragover', (e: DragEvent) => {
      e.preventDefault()
      if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy'
    })
    el.addEventListener('dragleave', (e: DragEvent) => {
      e.preventDefault()
      dragCounterRef.current--
      if (dragCounterRef.current <= 0) {
        dragCounterRef.current = 0
        setDragging(false)
      }
    })
    el.addEventListener('drop', (e: DragEvent) => {
      e.preventDefault()
      dragCounterRef.current = 0
      setDragging(false)
      const dt = e.dataTransfer
      // Try dataTransfer.files first (works on Firefox and most browsers)
      const file = dt?.files?.[0]
      if (file) {
        processFile(file)
        return
      }
      // Try dataTransfer.items for browsers that populate items but not files
      if (dt?.items) {
        for (let i = 0; i < dt.items.length; i++) {
          if (dt.items[i].kind === 'file') {
            const f = dt.items[i].getAsFile()
            if (f) { processFile(f); return }
          }
        }
      }
      // Chromium on Linux X11 may receive file drags as text/plain paths
      // instead of File objects. Auto-open the file picker as fallback.
      const textData = dt?.getData('text/plain')?.trim()
      if (textData && !dt?.types?.includes('Files')) {
        setError('Your browser received a file path instead of file data. Please use the file picker.')
        fileInputRef.current?.click()
      }
    })
  }, [processFile])

  const validateName = (name: string) => {
    if (!name) { setNameError('Name is required'); return false }
    if (name.length > 63) { setNameError('Name must be at most 63 characters'); return false }
    if (!DNS_LABEL_RE.test(name)) { setNameError('Must be lowercase alphanumeric and hyphens'); return false }
    setNameError(null)
    return true
  }

  const validateCvLabel = (label: string) => {
    if (!label) { setCvLabelError('Version label is required'); return false }
    if (!/[a-zA-Z0-9]/.test(label)) {
      setCvLabelError('Must contain at least one alphanumeric character')
      return false
    }
    setCvLabelError(null)
    return true
  }

  const buildRenameMap = () => {
    if (!prefix && !suffix) return undefined
    const data = fileData as { entity_types?: { name: string }[]; type_definitions?: { name: string }[] }
    const etRenames: Record<string, string> = {}
    const tdRenames: Record<string, string> = {}
    for (const et of data.entity_types || []) {
      etRenames[et.name] = `${prefix}${et.name}${suffix}`
    }
    for (const td of data.type_definitions || []) {
      tdRenames[td.name] = `${prefix}${td.name}${suffix}`
    }
    return { entity_types: etRenames, type_definitions: tdRenames }
  }

  const handleAnalyze = async () => {
    if (!fileData || !validateName(catalogName) || !validateCvLabel(cvLabel)) return
    setError(null)
    setLoading(true)
    try {
      const req = {
        catalog_name: catalogName || undefined,
        catalog_version_label: cvLabel || undefined,
        rename_map: buildRenameMap(),
        data: fileData,
      }
      const result = await api.catalogs.import(req, { dry_run: true }) as DryRunResponse
      setDryRunResult(result)
      if (result.collisions?.length > 0) {
        const autoReuse = result.collisions
          .filter(c => c.resolution === 'identical' && c.type !== 'catalog' && c.type !== 'catalog_version')
          .map(c => c.name)
        setReuseExisting(autoReuse)
        setStep('collisions')
      } else {
        setStep('confirm')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Dry run failed')
    } finally {
      setLoading(false)
    }
  }

  const handleImport = async () => {
    setError(null)
    setLoading(true)
    try {
      const req = {
        catalog_name: catalogName || undefined,
        catalog_version_label: cvLabel || undefined,
        rename_map: buildRenameMap(),
        reuse_existing: reuseExisting.length > 0 ? reuseExisting : undefined,
        data: fileData,
      }
      const result = await api.catalogs.import(req) as ImportResponse
      setImportResult(result)
      setStep('done')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Import failed')
    } finally {
      setLoading(false)
    }
  }

  const toggleReuse = (name: string) => {
    setReuseExisting(prev =>
      prev.includes(name) ? prev.filter(n => n !== name) : [...prev, name]
    )
  }

  return (
    <Modal isOpen={isOpen} onClose={handleClose} variant={ModalVariant.large}>
      <ModalHeader title={step === 'done' ? 'Import Complete' : 'Import Catalog'} />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

        {step === 'upload' && (
          <Form>
            <FormGroup label="Catalog File (JSON)" isRequired fieldId="import-file">
              <div
                ref={dropzoneCallbackRef}
                data-testid="file-dropzone"
                onClick={() => fileInputRef.current?.click()}
                style={{
                  border: `2px dashed ${dragging ? 'var(--pf-t--global--color--brand--default)' : '#ccc'}`,
                  borderRadius: '8px',
                  padding: '2rem',
                  textAlign: 'center',
                  cursor: 'pointer',
                  background: dragging ? 'var(--pf-t--global--color--brand--default--hover)' : undefined,
                  transition: 'border-color 0.2s, background 0.2s',
                }}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".json"
                  onChange={handleFileUpload}
                  data-testid="import-file-input"
                  style={{ display: 'none' }}
                />
                {fileName
                  ? <span>{fileName}</span>
                  : <span>{dragging ? 'Drop file here' : 'Click or drag a JSON file here'}</span>
                }
              </div>
            </FormGroup>
            {fileData !== null && (
              <>
                <FormGroup label="Catalog Name" fieldId="catalog-name" isRequired>
                  <TextInput
                    id="catalog-name"
                    value={catalogName}
                    onChange={(_e, v) => { setCatalogName(v); validateName(v) }}
                    validated={nameError ? 'error' : 'default'}
                  />
                  {nameError && <HelperText><HelperTextItem variant="error">{nameError}</HelperTextItem></HelperText>}
                </FormGroup>
                <FormGroup label="Catalog Version Label" fieldId="cv-label">
                  <TextInput
                    id="cv-label"
                    value={cvLabel}
                    onChange={(_e, v) => { setCvLabel(v); validateCvLabel(v) }}
                    validated={cvLabelError ? 'error' : 'default'}
                  />
                  {cvLabelError && <HelperText><HelperTextItem variant="error">{cvLabelError}</HelperTextItem></HelperText>}
                </FormGroup>
                <FormGroup label="Mass Rename Prefix" fieldId="prefix">
                  <TextInput id="prefix" value={prefix} onChange={(_e, v) => setPrefix(v)} placeholder="e.g. imported-" />
                </FormGroup>
                <FormGroup label="Mass Rename Suffix" fieldId="suffix">
                  <TextInput id="suffix" value={suffix} onChange={(_e, v) => setSuffix(v)} placeholder="e.g. -v2" />
                </FormGroup>
              </>
            )}
          </Form>
        )}

        {step === 'collisions' && dryRunResult && (
          <>
            <Alert
              variant={dryRunResult.summary.conflicts > 0 ? 'warning' : 'info'}
              title={`${dryRunResult.summary.total_entities} entities: ${dryRunResult.summary.new} new, ${dryRunResult.summary.identical} identical, ${dryRunResult.summary.conflicts} conflicts`}
              isInline
              style={{ marginBottom: '1rem' }}
            />
            <Table aria-label="Collision resolution" variant="compact">
              <Thead><Tr><Th>Name</Th><Th>Type</Th><Th>Status</Th><Th>Use existing</Th></Tr></Thead>
              <Tbody>
                {dryRunResult.collisions.map((c, i) => {
                  const canToggle = (c.resolution === 'identical' || c.resolution === 'conflict') &&
                    c.type !== 'catalog' && c.type !== 'catalog_version'
                  const isReusing = reuseExisting.includes(c.name)
                  return (
                    <Tr key={i}>
                      <Td>{c.name}</Td>
                      <Td>{c.type}</Td>
                      <Td>{c.resolution}{c.version ? ` (V${c.version})` : ''}</Td>
                      <Td>
                        {canToggle ? (
                          <>
                            <Checkbox
                              id={`reuse-${c.name}`}
                              isChecked={isReusing}
                              onChange={() => toggleReuse(c.name)}
                              label={isReusing ? 'Reuse existing' : 'Create new'}
                            />
                            {!isReusing && (prefix || suffix) && (
                              <span style={{ marginLeft: '1.5rem', color: 'var(--pf-t--global--text--color--subtle)' }}>
                                as &quot;{prefix}{c.name}{suffix}&quot;
                              </span>
                            )}
                            {!isReusing && !prefix && !suffix && (
                              <HelperText style={{ marginLeft: '1.5rem' }}>
                                <HelperTextItem variant="error">
                                  Name already exists — use existing or go back to set a prefix/suffix
                                </HelperTextItem>
                              </HelperText>
                            )}
                          </>
                        ) : c.resolution === 'conflict' ? (
                          <HelperText>
                            <HelperTextItem variant="error">
                              {c.type === 'catalog' ? 'Name already exists — go back and change the catalog name'
                                : c.type === 'catalog_version' ? 'Label already exists — go back and change the version label'
                                : 'Conflict — go back and change the name'}
                            </HelperTextItem>
                          </HelperText>
                        ) : (
                          <span>Create</span>
                        )}
                      </Td>
                    </Tr>
                  )
                })}
              </Tbody>
            </Table>
          </>
        )}

        {step === 'confirm' && dryRunResult && (
          <Alert
            variant="info"
            title={`Ready to import: ${dryRunResult.summary.new} new entities${dryRunResult.summary.identical > 0 ? `, ${dryRunResult.summary.identical} identical (will be reused)` : ''}.`}
            isInline
          />
        )}

        {step === 'done' && importResult && (
          <Alert
            variant="success"
            title={`Catalog "${importResult.catalog_name}" imported successfully`}
            isInline
          >
            <p>{importResult.types_created} types created, {importResult.types_reused} reused, {importResult.instances_created} instances created, {importResult.links_created} links created.</p>
          </Alert>
        )}

        {loading && <Spinner aria-label="Loading" style={{ marginTop: '1rem' }} />}
      </ModalBody>
      <ModalFooter>
        {step === 'upload' && (
          <>
            <Button variant="primary" onClick={handleAnalyze} isDisabled={!fileData || !!nameError || !!cvLabelError || loading}>
              Analyze
            </Button>
            <Button variant="link" onClick={handleClose}>Cancel</Button>
          </>
        )}
        {step === 'collisions' && (
          <>
            <Button
              variant="primary"
              onClick={() => setStep('confirm')}
              isDisabled={dryRunResult!.collisions.some(c =>
                c.resolution === 'conflict' && (c.type === 'catalog' || c.type === 'catalog_version')
              ) || dryRunResult!.collisions.some(c =>
                (c.resolution === 'identical' || c.resolution === 'conflict') &&
                c.type !== 'catalog' && c.type !== 'catalog_version' &&
                !reuseExisting.includes(c.name) &&
                !prefix && !suffix
              )}
            >
              Continue
            </Button>
            <Button variant="link" onClick={() => setStep('upload')}>Back</Button>
          </>
        )}
        {step === 'confirm' && (
          <>
            <Button variant="primary" onClick={handleImport} isDisabled={loading} isLoading={loading}>
              Import
            </Button>
            <Button variant="link" onClick={() => setStep(dryRunResult?.collisions?.length ? 'collisions' : 'upload')}>Back</Button>
          </>
        )}
        {step === 'done' && (
          <Button variant="primary" onClick={() => { handleClose(); onSuccess(importResult!.catalog_name) }}>
            View Catalog
          </Button>
        )}
      </ModalFooter>
    </Modal>
  )
}
