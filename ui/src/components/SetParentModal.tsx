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
import type { EntityInstance } from '../types'

interface Props {
  isOpen: boolean
  onClose: () => void
  instanceName: string | undefined
  parentTypeName: string
  parentInstances: EntityInstance[]
  hasParent: boolean
  onSubmit: (parentType: string, parentId: string) => Promise<void>
  onRemoveParent: () => void
  error: string | null
}

export default function SetParentModal({
  isOpen, onClose,
  instanceName,
  parentTypeName,
  parentInstances,
  hasParent,
  onSubmit, onRemoveParent,
  error,
}: Props) {
  const [parentInstanceId, setParentInstanceId] = useState('')
  const [parentInstSelectOpen, setParentInstSelectOpen] = useState(false)

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setParentInstanceId('')
    }
  }, [isOpen])

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title={`Set Container for ${instanceName}`} />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Container Type" fieldId="parent-type">
            <TextInput id="parent-type" value={parentTypeName} isDisabled aria-label="Container type" />
          </FormGroup>
          <FormGroup label="Container Instance" isRequired fieldId="parent-instance">
            <Select
              id="parent-instance"
              isOpen={parentInstSelectOpen}
              selected={parentInstanceId}
              onSelect={(_e, val) => { setParentInstanceId(val as string); setParentInstSelectOpen(false) }}
              onOpenChange={setParentInstSelectOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setParentInstSelectOpen(!parentInstSelectOpen)} isExpanded={parentInstSelectOpen} style={{ width: '100%' }}>
                  {parentInstances.find(i => i.id === parentInstanceId)?.name || 'Select container...'}
                </MenuToggle>
              )}
            >
              {parentInstances.map(inst => (
                <SelectOption key={inst.id} value={inst.id}>{inst.name}</SelectOption>
              ))}
            </Select>
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => onSubmit(parentTypeName, parentInstanceId)} isDisabled={!parentTypeName || !parentInstanceId}>Set Container</Button>
        <Button variant="danger" onClick={onRemoveParent} isDisabled={!hasParent}>Remove Container</Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
