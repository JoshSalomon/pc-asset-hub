import { useState, useEffect } from 'react'
import {
  Modal, ModalBody, ModalHeader, ModalFooter, ModalVariant,
  Button, Alert, AlertVariant, Label, Spinner,
} from '@patternfly/react-core'
import { api } from '../api/client'

interface BindingResult {
  binding_id: string
  exporter_name: string
  status: string
  artifact_count: number
  error: string
}

interface PreviewResult {
  session_token: string
  expires_at: string
  bindings: BindingResult[]
  has_failures: boolean
}

interface Props {
  catalogName: string
  onClose: () => void
  onPublished: () => void
}

export default function PublishPreviewModal({ catalogName, onClose, onPublished }: Props) {
  const [preview, setPreview] = useState<PreviewResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [publishing, setPublishing] = useState(false)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const result = await api.catalogs.publishPreview(catalogName)
        if (!cancelled) {
          setPreview(result)
          setLoading(false)
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to generate preview')
          setLoading(false)
        }
      }
    })()
    return () => { cancelled = true }
  }, [catalogName])

  const handlePublish = async () => {
    if (!preview) return
    try {
      setPublishing(true)
      setError(null)
      await api.catalogs.publishWithToken(catalogName, preview.session_token)
      await Promise.all(
        preview.bindings
          .filter(b => b.status === 'success')
          .map(b => api.exportBindings.download(catalogName, preview.session_token, b.binding_id))
      )
      onPublished()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Publish failed')
      setPublishing(false)
    }
  }

  return (
    <Modal variant={ModalVariant.medium} isOpen onClose={onClose}>
      <ModalHeader title="Publish Preview" />
      <ModalBody>
        {error && <Alert variant={AlertVariant.danger} title={error} isInline style={{ marginBottom: '1rem' }} />}
        {loading && <Spinner size="lg" />}
        {preview && (
          <>
            {preview.has_failures && (
              <Alert variant={AlertVariant.warning} title="Some export bindings failed" isInline style={{ marginBottom: '1rem' }} />
            )}
            <table className="pf-v6-c-table pf-m-grid-md" role="grid" aria-label="Preview results">
              <thead>
                <tr role="row">
                  <th role="columnheader">Exporter</th>
                  <th role="columnheader">Status</th>
                  <th role="columnheader">Artifacts</th>
                  <th role="columnheader">Error</th>
                </tr>
              </thead>
              <tbody>
                {preview.bindings.map(b => (
                  <tr key={b.binding_id} role="row">
                    <td role="gridcell">{b.exporter_name}</td>
                    <td role="gridcell">
                      <Label color={b.status === 'success' ? 'green' : 'red'}>{b.status}</Label>
                    </td>
                    <td role="gridcell">{b.artifact_count}</td>
                    <td role="gridcell">{b.error}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </ModalBody>
      <ModalFooter>
        {preview && !preview.has_failures && (
          <Button variant="primary" onClick={handlePublish} isLoading={publishing} isDisabled={publishing}>
            Publish
          </Button>
        )}
        {preview && preview.has_failures && (
          <Button variant="warning" onClick={handlePublish} isLoading={publishing} isDisabled={publishing}>
            Publish Anyway
          </Button>
        )}
        <Button variant="link" onClick={onClose} isDisabled={publishing}>
          {preview?.has_failures ? 'Abort' : 'Cancel'}
        </Button>
      </ModalFooter>
    </Modal>
  )
}
