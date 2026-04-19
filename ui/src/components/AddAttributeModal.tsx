import { useState, useEffect, useCallback } from 'react'
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
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core'
import type { TypeDefinition } from '../types'
import TypeDefinitionSelector from './TypeDefinitionSelector'

export interface AddAttributeValues {
  name: string
  description: string
  typeDefinitionVersionId: string
  required: boolean
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (values: AddAttributeValues) => Promise<void>
  typeDefinitions: TypeDefinition[]
  error: string | null
}

export default function AddAttributeModal({ isOpen, onClose, onSubmit, typeDefinitions, error }: Props) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedTdId, setSelectedTdId] = useState('')
  const [required, setRequired] = useState(false)

  const resetForm = useCallback(() => {
    setName('')
    setDescription('')
    setSelectedTdId('')
    setRequired(false)
  }, [])

  useEffect(() => {
    if (isOpen) resetForm()
  }, [isOpen, resetForm])

  const handleClose = () => {
    resetForm()
    onClose()
  }

  const handleSubmit = async () => {
    const td = typeDefinitions.find(t => t.id === selectedTdId)
    if (!td || !td.latest_version_id) return
    await onSubmit({ name, description, typeDefinitionVersionId: td.latest_version_id, required })
  }

  const selectedTd = typeDefinitions.find(t => t.id === selectedTdId)

  return (
    <Modal variant={ModalVariant.small} isOpen={isOpen} onClose={handleClose}>
      <ModalHeader title="Add Attribute" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Name" isRequired fieldId="attr-name">
            <TextInput id="attr-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
          </FormGroup>
          <FormGroup label="Description" fieldId="attr-desc">
            <TextInput id="attr-desc" value={description} onChange={(_e, v) => setDescription(v)} />
          </FormGroup>
          <FormGroup label="Type" isRequired fieldId="attr-type">
            <TypeDefinitionSelector
              typeDefinitions={typeDefinitions}
              selectedTdId={selectedTdId}
              onSelect={setSelectedTdId}
            />
            {selectedTd && selectedTd.base_type === 'string' && (
              <HelperText>
                <HelperTextItem>For multiline text, set the "multiline" constraint in the type definition.</HelperTextItem>
              </HelperText>
            )}
          </FormGroup>
          <FormGroup fieldId="attr-required">
            <label>
              <input type="checkbox" id="attr-required" checked={required} onChange={(e) => setRequired(e.target.checked)} />
              {' '}Required
            </label>
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSubmit} isDisabled={!name.trim() || !selectedTdId}>Add</Button>
        <Button variant="link" onClick={handleClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
