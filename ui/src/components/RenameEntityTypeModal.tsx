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

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (newName: string, deepCopyAllowed: boolean) => Promise<void>
  currentName: string
  error: string | null
  // Deep copy warning
  deepCopyWarningOpen: boolean
  pendingNewName: string
  onDeepCopyConfirm: () => void
  onDeepCopyCancel: () => void
}

export default function RenameEntityTypeModal({
  isOpen, onClose, onSubmit, currentName, error,
  deepCopyWarningOpen, pendingNewName, onDeepCopyConfirm, onDeepCopyCancel,
}: Props) {
  const [name, setName] = useState(currentName)

  useEffect(() => {
    if (isOpen) {
      setName(currentName)
    }
  }, [isOpen, currentName])

  const handleClose = () => {
    onClose()
  }

  const handleSubmit = async () => {
    await onSubmit(name.trim(), false)
  }

  return (
    <>
      {/* Rename Modal */}
      <Modal variant={ModalVariant.small} isOpen={isOpen} onClose={handleClose}>
        <ModalHeader title="Rename Entity Type" />
        <ModalBody>
          {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="New Name" isRequired fieldId="edit-name">
              <TextInput id="edit-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleSubmit} isDisabled={!name.trim() || name.trim() === currentName}>Rename</Button>
          <Button variant="link" onClick={handleClose}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Deep Copy Warning Modal */}
      <Modal variant={ModalVariant.small} isOpen={deepCopyWarningOpen} onClose={onDeepCopyCancel}>
        <ModalHeader title="Deep Copy Required" />
        <ModalBody>
          <Alert variant="warning" title="This entity type is referenced by catalog versions in testing/production." isInline style={{ marginBottom: '1rem' }} />
          Renaming will create a new entity type with the name &quot;{pendingNewName}&quot;. The original entity type will remain unchanged in existing catalog versions.
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={onDeepCopyConfirm}>Create Copy</Button>
          <Button variant="link" onClick={onDeepCopyCancel}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </>
  )
}
