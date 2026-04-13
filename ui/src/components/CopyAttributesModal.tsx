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
import type { EntityType, Attribute } from '../types'

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (selectedAttrs: string[]) => Promise<void>
  onLoadSource: (sourceId: string) => Promise<void>
  entityTypes: EntityType[]
  currentEntityTypeId: string | undefined
  sourceAttributes: Attribute[]
  existingAttributes: Attribute[]
  error: string | null
}

function typeLabel(attr: Attribute): string {
  if (attr.type_name) return attr.type_name
  if (attr.base_type) return attr.base_type
  return 'unknown'
}

function typeLabelColor(attr: Attribute): 'purple' | 'blue' | 'green' | 'grey' {
  const bt = attr.base_type || ''
  if (bt === 'enum') return 'purple'
  if (bt === 'integer' || bt === 'number') return 'blue'
  if (bt === 'boolean') return 'green'
  return 'grey'
}

export default function CopyAttributesModal({
  isOpen, onClose, onSubmit, onLoadSource,
  entityTypes, currentEntityTypeId,
  sourceAttributes, existingAttributes,
  error,
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
                      <Label color={typeLabelColor(sa)}>
                        {typeLabel(sa)}
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
