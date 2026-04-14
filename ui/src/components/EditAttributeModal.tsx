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
  SelectGroup,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'
import type { TypeDefinition } from '../types'

export interface EditAttributeValues {
  name: string
  description: string
  typeDefinitionVersionId: string
  required: boolean
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (values: EditAttributeValues) => Promise<void>
  typeDefinitions: TypeDefinition[]
  error: string | null
  initialName: string
  initialDescription: string
  initialTypeDefinitionId: string
  initialRequired: boolean
}

export default function EditAttributeModal({
  isOpen, onClose, onSubmit, typeDefinitions, error,
  initialName, initialDescription, initialTypeDefinitionId, initialRequired,
}: Props) {
  const [name, setName] = useState(initialName)
  const [description, setDescription] = useState(initialDescription)
  const [selectedTdId, setSelectedTdId] = useState(initialTypeDefinitionId)
  const [tdOpen, setTdOpen] = useState(false)
  const [required, setRequired] = useState(initialRequired)

  // Reset form when modal opens with new initial values.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (isOpen) {
      setName(initialName)
      setDescription(initialDescription)
      setSelectedTdId(initialTypeDefinitionId)
      setRequired(initialRequired)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  const handleClose = () => {
    onClose()
  }

  const handleSubmit = async () => {
    const td = typeDefinitions.find(t => t.id === selectedTdId)
    if (!td) return
    await onSubmit({ name, description, typeDefinitionVersionId: td.latest_version_id, required })
  }

  const systemTypes = typeDefinitions.filter(td => td.system)
  const customTypes = typeDefinitions.filter(td => !td.system)

  const selectedTd = typeDefinitions.find(t => t.id === selectedTdId)
  const toggleLabel = selectedTd ? `${selectedTd.name} (${selectedTd.base_type})` : 'Select type...'

  return (
    <Modal variant={ModalVariant.small} isOpen={isOpen} onClose={handleClose}>
      <ModalHeader title="Edit Attribute" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Name" isRequired fieldId="edit-attr-name">
            <TextInput id="edit-attr-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
          </FormGroup>
          <FormGroup label="Description" fieldId="edit-attr-desc">
            <TextInput id="edit-attr-desc" value={description} onChange={(_e, v) => setDescription(v)} />
          </FormGroup>
          <FormGroup label="Type" isRequired fieldId="edit-attr-type">
            <Select
              isOpen={tdOpen}
              selected={selectedTdId}
              onSelect={(_e, value) => { setSelectedTdId(value as string); setTdOpen(false) }}
              onOpenChange={setTdOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setTdOpen(!tdOpen)} isExpanded={tdOpen}>
                  {toggleLabel}
                </MenuToggle>
              )}
            >
              {systemTypes.length > 0 && (
                <SelectGroup label="System Types">
                  {systemTypes.map(td => (
                    <SelectOption key={td.id} value={td.id}>
                      {td.name} ({td.base_type})
                    </SelectOption>
                  ))}
                </SelectGroup>
              )}
              {customTypes.length > 0 && (
                <SelectGroup label="Custom Types">
                  {customTypes.map(td => (
                    <SelectOption key={td.id} value={td.id}>
                      {td.name} ({td.base_type})
                    </SelectOption>
                  ))}
                </SelectGroup>
              )}
            </Select>
          </FormGroup>
          <FormGroup fieldId="edit-attr-required">
            <label>
              <input type="checkbox" id="edit-attr-required" checked={required} onChange={(e) => setRequired(e.target.checked)} />
              {' '}Required
            </label>
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSubmit} isDisabled={!name.trim()}>Save</Button>
        <Button variant="link" onClick={handleClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
