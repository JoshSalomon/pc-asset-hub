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
import type { Enum } from '../types'

export interface EditAttributeValues {
  name: string
  description: string
  type: string
  enumId: string
  required: boolean
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (values: EditAttributeValues) => Promise<void>
  enums: Enum[]
  error: string | null
  initialName: string
  initialDescription: string
  initialType: string
  initialEnumId: string
  initialRequired: boolean
}

export default function EditAttributeModal({
  isOpen, onClose, onSubmit, enums, error,
  initialName, initialDescription, initialType, initialEnumId, initialRequired,
}: Props) {
  const [name, setName] = useState(initialName)
  const [description, setDescription] = useState(initialDescription)
  const [type, setType] = useState(initialType)
  const [typeOpen, setTypeOpen] = useState(false)
  const [enumId, setEnumId] = useState(initialEnumId)
  const [enumOpen, setEnumOpen] = useState(false)
  const [required, setRequired] = useState(initialRequired)

  // Reset form when modal opens with new initial values.
  // eslint-disable-next-line react-hooks/exhaustive-deps — intentionally depend only on isOpen
  // to avoid mid-edit resets if the parent re-renders with updated initial* props.
  useEffect(() => {
    if (isOpen) {
      setName(initialName)
      setDescription(initialDescription)
      setType(initialType)
      setEnumId(initialEnumId)
      setRequired(initialRequired)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  const handleClose = () => {
    onClose()
  }

  const handleSubmit = async () => {
    await onSubmit({ name, description, type, enumId, required })
  }

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
              isOpen={typeOpen}
              selected={type}
              onSelect={(_e, value) => { setType(value as string); setTypeOpen(false) }}
              onOpenChange={setTypeOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setTypeOpen(!typeOpen)} isExpanded={typeOpen}>{type}</MenuToggle>
              )}
            >
              <SelectOption value="string">string</SelectOption>
              <SelectOption value="number">number</SelectOption>
              <SelectOption value="enum">enum</SelectOption>
            </Select>
          </FormGroup>
          {type === 'enum' && (
            <FormGroup label="Enum" isRequired fieldId="edit-attr-enum">
              <Select
                isOpen={enumOpen}
                selected={enumId}
                onSelect={(_e, value) => { setEnumId(value as string); setEnumOpen(false) }}
                onOpenChange={setEnumOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setEnumOpen(!enumOpen)} isExpanded={enumOpen}>
                    {enums.find((en) => en.id === enumId)?.name || 'Select enum'}
                  </MenuToggle>
                )}
              >
                {enums.map((en) => (
                  <SelectOption key={en.id} value={en.id}>{en.name}</SelectOption>
                ))}
              </Select>
            </FormGroup>
          )}
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
