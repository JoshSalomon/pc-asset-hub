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
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'
import type { Catalog } from '../types'
import { isValidDnsLabel } from '../utils/dnsLabel'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (target: string, archiveName: string) => Promise<void>
  availableCatalogs: Catalog[]
  error: string | null
  loading: boolean
}

export default function ReplaceCatalogModal({
  isOpen, onClose, onSubmit, availableCatalogs, error, loading,
}: Props) {
  const [target, setTarget] = useState('')
  const [archiveName, setArchiveName] = useState('')
  const [targetSelectOpen, setTargetSelectOpen] = useState(false)

  useEffect(() => {
    if (isOpen) {
      setTarget('')
      setArchiveName('')
    }
  }, [isOpen])

  return (
    <Modal
      variant={ModalVariant.small}
      isOpen={isOpen}
      onClose={onClose}
    >
      <ModalHeader title="Replace Catalog" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Target catalog" isRequired fieldId="replace-target">
            <Select
              id="replace-target"
              isOpen={targetSelectOpen}
              selected={target}
              onSelect={(_e, val) => { setTarget(val as string); setTargetSelectOpen(false) }}
              onOpenChange={setTargetSelectOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setTargetSelectOpen(!targetSelectOpen)} isExpanded={targetSelectOpen} style={{ width: '100%' }}>
                  {target || 'Select target catalog...'}
                </MenuToggle>
              )}
            >
              {availableCatalogs.map(c => (
                <SelectOption key={c.name} value={c.name}>{c.name}</SelectOption>
              ))}
            </Select>
          </FormGroup>
          <FormGroup label="Archive name (optional)" fieldId="replace-archive">
            <TextInput id="replace-archive" value={archiveName} onChange={(_e, v) => setArchiveName(v)}
              placeholder={target ? `${target}-archive-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}` : ''}
              validated={archiveName && !isValidDnsLabel(archiveName) ? 'error' : 'default'}
            />
            {archiveName && !isValidDnsLabel(archiveName) && (
              <div style={{ color: '#c9190b', fontSize: '0.875rem', marginTop: '0.25rem' }}>Must be a valid DNS label</div>
            )}
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" isDisabled={!target || (!!archiveName && !isValidDnsLabel(archiveName)) || loading} isLoading={loading}
          onClick={() => onSubmit(target, archiveName)}>
          Replace
        </Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
