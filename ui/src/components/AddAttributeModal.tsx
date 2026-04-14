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
  const [tdOpen, setTdOpen] = useState(false)
  const [required, setRequired] = useState(false)

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setName('')
      setDescription('')
      setSelectedTdId('')
      setRequired(false)
    }
  }, [isOpen])

  const handleClose = () => {
    setName('')
    setDescription('')
    setSelectedTdId('')
    setRequired(false)
    onClose()
  }

  const handleSubmit = async () => {
    const td = typeDefinitions.find(t => t.id === selectedTdId)
    if (!td || !td.latest_version_id) return
    await onSubmit({ name, description, typeDefinitionVersionId: td.latest_version_id, required })
  }

  const systemTypes = typeDefinitions.filter(td => td.system)
  const customTypes = typeDefinitions.filter(td => !td.system)

  const selectedTd = typeDefinitions.find(t => t.id === selectedTdId)
  const toggleLabel = selectedTd ? `${selectedTd.name} (${selectedTd.base_type})` : 'Select type...'

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
