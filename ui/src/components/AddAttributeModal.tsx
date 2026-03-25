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

export interface AddAttributeValues {
  name: string
  description: string
  type: string
  enumId: string
  required: boolean
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (values: AddAttributeValues) => Promise<void>
  enums: Enum[]
  error: string | null
}

export default function AddAttributeModal({ isOpen, onClose, onSubmit, enums, error }: Props) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [type, setType] = useState('string')
  const [typeOpen, setTypeOpen] = useState(false)
  const [enumId, setEnumId] = useState('')
  const [enumOpen, setEnumOpen] = useState(false)
  const [required, setRequired] = useState(false)

  // Reset form when modal opens (handles the case where isOpen is toggled
  // without going through handleClose, e.g. after a successful submit)
  useEffect(() => {
    if (isOpen) {
      setName('')
      setDescription('')
      setType('string')
      setEnumId('')
      setRequired(false)
    }
  }, [isOpen])

  const handleClose = () => {
    setName('')
    setDescription('')
    setType('string')
    setEnumId('')
    setRequired(false)
    onClose()
  }

  const handleSubmit = async () => {
    await onSubmit({ name, description, type, enumId, required })
  }

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
            <FormGroup label="Enum" isRequired fieldId="attr-enum">
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
          <FormGroup fieldId="attr-required">
            <label>
              <input type="checkbox" id="attr-required" checked={required} onChange={(e) => setRequired(e.target.checked)} />
              {' '}Required
            </label>
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSubmit} isDisabled={!name.trim()}>Add</Button>
        <Button variant="link" onClick={handleClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
