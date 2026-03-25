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
import type { EntityInstance, SnapshotAttribute } from '../types'
import AttributeFormFields from './AttributeFormFields'

// This modal uses `instance !== null` as its open/close signal rather than a
// separate `isOpen` prop. The instance itself carries the data needed to
// pre-populate the form, so a null instance means "closed" and a non-null
// instance means "open with this data". This differs from other modals that
// use an explicit `isOpen` boolean.
interface Props {
  instance: EntityInstance | null  // null = closed
  onClose: () => void
  schemaAttrs: SnapshotAttribute[]
  enumValues: Record<string, string[]>
  onSubmit: (version: number, name: string, description: string, attrs: Record<string, string>) => Promise<void>
  error: string | null
}

export default function EditInstanceModal({
  instance, onClose,
  schemaAttrs, enumValues,
  onSubmit, error,
}: Props) {
  const [name, setName] = useState('')
  const [desc, setDesc] = useState('')
  const [attrs, setAttrs] = useState<Record<string, string>>({})

  // Pre-populate from instance when modal opens
  useEffect(() => {
    if (instance) {
      setName(instance.name)
      setDesc(instance.description)
      const a: Record<string, string> = {}
      for (const av of instance.attributes || []) {
        if (av.system) continue
        a[av.name] = av.value != null ? String(av.value) : ''
      }
      setAttrs(a)
    }
  }, [instance])

  return (
    <Modal variant={ModalVariant.medium} isOpen={instance !== null} onClose={onClose}>
      <ModalHeader title={`Edit ${instance?.name}`} />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <AttributeFormFields
            schemaAttrs={schemaAttrs}
            values={attrs}
            onChange={(n, v) => setAttrs(prev => ({ ...prev, [n]: v }))}
            enumValues={enumValues}
            idPrefix="edit"
            includeSystem
            systemName={name}
            setSystemName={setName}
            systemDesc={desc}
            setSystemDesc={setDesc}
          />
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => instance && onSubmit(instance.version, name, desc, attrs)}>Save</Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
