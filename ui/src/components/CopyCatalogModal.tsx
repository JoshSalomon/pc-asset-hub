import { useState, useEffect } from 'react'
import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Form,
  FormGroup,
  TextInput,
  Button,
  Alert,
} from '@patternfly/react-core'
import { isValidDnsLabel } from '../utils/dnsLabel'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (name: string, description: string) => Promise<void>
  error: string | null
  loading: boolean
}

export default function CopyCatalogModal({
  isOpen, onClose, onSubmit, error, loading,
}: Props) {
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')

  useEffect(() => {
    if (isOpen) {
      setName('')
      setDesc('')
    }
  }, [isOpen])

  return (
    <Modal
      variant={ModalVariant.small}
      isOpen={isOpen}
      onClose={onClose}
    >
      <ModalHeader title="Copy Catalog" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="New catalog name" isRequired fieldId="copy-name">
            <TextInput id="copy-name" value={name} onChange={(_e, v) => setName(v)}
              validated={name && !isValidDnsLabel(name) ? 'error' : 'default'}
            />
            {name && !isValidDnsLabel(name) && (
              <div style={{ color: '#c9190b', fontSize: '0.875rem', marginTop: '0.25rem' }}>Must be a valid DNS label (lowercase alphanumeric and hyphens)</div>
            )}
          </FormGroup>
          <FormGroup label="Description" fieldId="copy-desc">
            <TextInput id="copy-desc" value={desc} onChange={(_e, v) => setDesc(v)} />
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" isDisabled={!name || !isValidDnsLabel(name) || loading} isLoading={loading}
          onClick={() => onSubmit(name, desc)}>
          Copy
        </Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
