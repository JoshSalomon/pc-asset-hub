import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Form,
  FormGroup,
  Button,
  Alert,
  Label,
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { useState, useEffect } from 'react'
import type { EntityType, Attribute, Enum } from '../types'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (selectedAttrs: string[]) => Promise<void>
  onLoadSource: (sourceId: string) => Promise<void>
  entityTypes: EntityType[]
  currentEntityTypeId: string | undefined
  sourceAttributes: Attribute[]
  existingAttributes: Attribute[]
  enums: Enum[]
  error: string | null
}

export default function CopyAttributesModal({
  isOpen, onClose, onSubmit, onLoadSource,
  entityTypes, currentEntityTypeId,
  sourceAttributes, existingAttributes,
  enums, error,
}: Props) {
  const [sourceOpen, setSourceOpen] = useState(false)
  const [sourceId, setSourceId] = useState('')
  const [selectedCopyAttrs, setSelectedCopyAttrs] = useState<string[]>([])

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setSourceId('')
      setSelectedCopyAttrs([])
    }
  }, [isOpen])

  const handleClose = () => {
    setSourceId('')
    setSelectedCopyAttrs([])
    onClose()
  }

  const handleSelectSource = (id: string) => {
    setSourceId(id)
    setSourceOpen(false)
    setSelectedCopyAttrs([])
    onLoadSource(id)
  }

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={handleClose}>
      <ModalHeader title="Copy Attributes from Another Type" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Source Entity Type" isRequired fieldId="copy-attrs-source">
            <Select
              isOpen={sourceOpen}
              selected={sourceId}
              onSelect={(_e, value) => handleSelectSource(value as string)}
              onOpenChange={setSourceOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setSourceOpen(!sourceOpen)} isExpanded={sourceOpen}>
                  {entityTypes.find((et) => et.id === sourceId)?.name || 'Select source type'}
                </MenuToggle>
              )}
            >
              {entityTypes.filter((et) => et.id !== currentEntityTypeId).map((et) => (
                <SelectOption key={et.id} value={et.id}>{et.name}</SelectOption>
              ))}
            </Select>
          </FormGroup>
        </Form>
        {sourceAttributes.length > 0 && (
          <Table aria-label="Source attributes" style={{ marginTop: '1rem' }}>
            <Thead>
              <Tr>
                <Th />
                <Th>Name</Th>
                <Th>Type</Th>
                <Th>Description</Th>
                <Th>Status</Th>
              </Tr>
            </Thead>
            <Tbody>
              {sourceAttributes.filter((sa) => !sa.system).map((sa) => {
                const conflict = existingAttributes.some((a) => a.name === sa.name)
                return (
                  <Tr key={sa.id}>
                    <Td>
                      <input
                        type="checkbox"
                        disabled={conflict}
                        checked={selectedCopyAttrs.includes(sa.name)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedCopyAttrs((prev) => [...prev, sa.name])
                          } else {
                            setSelectedCopyAttrs((prev) => prev.filter((n) => n !== sa.name))
                          }
                        }}
                      />
                    </Td>
                    <Td>{sa.name}{sa.required ? ' *' : ''}</Td>
                    <Td>
                      <Label color={sa.type === 'enum' ? 'purple' : sa.type === 'number' ? 'blue' : 'grey'}>
                        {sa.type === 'enum' && (sa.enum_name || sa.enum_id)
                          ? `enum (${sa.enum_name || enums.find((en) => en.id === sa.enum_id)?.name || sa.enum_id?.slice(0, 8)})`
                          : sa.type}
                      </Label>
                    </Td>
                    <Td>{sa.description || '-'}</Td>
                    <Td>{conflict ? <Label color="red">Conflict</Label> : <Label color="green">Available</Label>}</Td>
                  </Tr>
                )
              })}
            </Tbody>
          </Table>
        )}
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => onSubmit(selectedCopyAttrs)} isDisabled={selectedCopyAttrs.length === 0}>Copy Selected</Button>
        <Button variant="link" onClick={handleClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
