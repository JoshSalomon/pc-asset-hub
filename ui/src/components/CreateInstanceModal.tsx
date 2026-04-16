import { useState, useEffect } from 'react'
import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Form,
  Button,
  Alert,
} from '@patternfly/react-core'
import type { SnapshotAttribute } from '../types'
import AttributeFormFields from './AttributeFormFields'

interface Props {
  isOpen: boolean
  onClose: () => void
  entityTypeName: string
  schemaAttrs: SnapshotAttribute[]
  enumValues: Record<string, string[]>
  onSubmit: (name: string, description: string, attrs: Record<string, string>) => Promise<void>
  error: string | null
}

export default function CreateInstanceModal({
  isOpen, onClose, entityTypeName,
  schemaAttrs, enumValues,
  onSubmit, error,
}: Props) {
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [attrs, setAttrs] = useState<Record<string, string>>({})

  // Reset form when modal opens; initialize mandatory booleans to "false"
  useEffect(() => {
    if (isOpen) {
      setName('')
      setDesc('')
      const initial: Record<string, string> = {}
      for (const attr of schemaAttrs) {
        if (!attr.system && attr.base_type === 'boolean' && attr.required) {
          initial[attr.name] = 'false'
        }
      }
      setAttrs(initial)
    }
  }, [isOpen, schemaAttrs])

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title={`Create ${entityTypeName}`} />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <AttributeFormFields
            schemaAttrs={schemaAttrs}
            values={attrs}
            onChange={(n, v) => setAttrs(prev => ({ ...prev, [n]: v }))}
            enumValues={enumValues}
            idPrefix="inst"
            includeSystem
            systemName={name}
            setSystemName={setName}
            systemDesc={desc}
            setSystemDesc={setDesc}
          />
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => onSubmit(name, desc, attrs)} isDisabled={!name.trim()}>Create</Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
