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
import type { EntityInstance, SnapshotAssociation, SnapshotAttribute } from '../types'
import AttributeFormFields from './AttributeFormFields'

export interface AddChildCreateData {
  name: string
  description: string
  attrs: Record<string, string>
}

export interface AddChildAdoptData {
  adoptInstanceId: string
}

interface Props {
  isOpen: boolean
  onClose: () => void
  schemaAssocs: SnapshotAssociation[]
  childSchemaAttrs: SnapshotAttribute[]
  childEnumValues: Record<string, string[]>
  availableInstances: EntityInstance[]
  onChildTypeChange: (typeName: string) => void
  onSubmit: (childType: string, mode: 'create' | 'adopt', data: AddChildCreateData | AddChildAdoptData) => Promise<void>
  error: string | null
  initialChildType?: string
}

export default function AddChildModal({
  isOpen, onClose,
  schemaAssocs,
  childSchemaAttrs, childEnumValues,
  availableInstances,
  onChildTypeChange,
  onSubmit, error,
  initialChildType,
}: Props) {
  const [childTypeName, setChildTypeName] = useState('')
  const [addChildMode, setAddChildMode] = useState<'create' | 'adopt'>('create')
  const [newChildName, setNewChildName] = useState('')
  const [newChildDesc, setNewChildDesc] = useState('')
  const [newChildAttrs, setNewChildAttrs] = useState<Record<string, string>>({})
  const [adoptInstanceId, setAdoptInstanceId] = useState('')

  const [childTypeSelectOpen, setChildTypeSelectOpen] = useState(false)
  const [adoptSelectOpen, setAdoptSelectOpen] = useState(false)
  const [modeSelectOpen, setModeSelectOpen] = useState(false)

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setChildTypeName(initialChildType || '')
      setAddChildMode('create')
      setNewChildName('')
      setNewChildDesc('')
      setNewChildAttrs({})
      setAdoptInstanceId('')
    }
  }, [isOpen, initialChildType])

  const handleSubmit = async () => {
    if (!childTypeName) return
    if (addChildMode === 'create') {
      await onSubmit(childTypeName, 'create', {
        name: newChildName,
        description: newChildDesc,
        attrs: newChildAttrs,
      })
    } else {
      await onSubmit(childTypeName, 'adopt', { adoptInstanceId })
    }
  }

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Add Contained Instance" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Child Entity Type" isRequired fieldId="child-type">
            <Select
              id="child-type"
              isOpen={childTypeSelectOpen}
              selected={childTypeName}
              onSelect={(_e, val) => {
                const v = val as string
                setChildTypeName(v)
                setChildTypeSelectOpen(false)
                setNewChildAttrs({})
                onChildTypeChange(v)
              }}
              onOpenChange={setChildTypeSelectOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setChildTypeSelectOpen(!childTypeSelectOpen)} isExpanded={childTypeSelectOpen} style={{ width: '100%' }}>
                  {childTypeName || 'Select child type...'}
                </MenuToggle>
              )}
            >
              {schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing').map(a => (
                <SelectOption key={a.target_entity_type_name} value={a.target_entity_type_name}>
                  {a.target_entity_type_name}
                </SelectOption>
              ))}
            </Select>
          </FormGroup>
          <FormGroup label="Mode" fieldId="child-mode">
            {availableInstances.length > 0 ? (
              <Select
                id="child-mode"
                isOpen={modeSelectOpen}
                selected={addChildMode}
                onSelect={(_e, val) => { setAddChildMode(val as 'create' | 'adopt'); setModeSelectOpen(false) }}
                onOpenChange={setModeSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setModeSelectOpen(!modeSelectOpen)} isExpanded={modeSelectOpen} style={{ width: '100%' }}>
                    {addChildMode === 'create' ? 'Create New' : 'Adopt Existing'}
                  </MenuToggle>
                )}
              >
                <SelectOption value="create">Create New</SelectOption>
                <SelectOption value="adopt">Adopt Existing</SelectOption>
              </Select>
            ) : (
              <MenuToggle isDisabled style={{ width: '100%' }}>Create New</MenuToggle>
            )}
          </FormGroup>
          {(addChildMode === 'create' || availableInstances.length === 0) ? (
            <>
              <FormGroup label="Name" isRequired fieldId="child-name">
                <TextInput id="child-name" value={newChildName} onChange={(_e, v) => setNewChildName(v)} isRequired />
              </FormGroup>
              <FormGroup label="Description" fieldId="child-desc">
                <TextInput id="child-desc" value={newChildDesc} onChange={(_e, v) => setNewChildDesc(v)} />
              </FormGroup>
              <AttributeFormFields
                schemaAttrs={childSchemaAttrs}
                values={newChildAttrs}
                onChange={(name, value) => setNewChildAttrs(prev => ({ ...prev, [name]: value }))}
                enumValues={childEnumValues}
                idPrefix="child"
                includeSystem={false}
              />
            </>
          ) : (
            <FormGroup label="Select Instance" isRequired fieldId="adopt-instance">
              <Select
                id="adopt-instance"
                isOpen={adoptSelectOpen}
                selected={adoptInstanceId}
                onSelect={(_e, val) => { setAdoptInstanceId(val as string); setAdoptSelectOpen(false) }}
                onOpenChange={setAdoptSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setAdoptSelectOpen(!adoptSelectOpen)} isExpanded={adoptSelectOpen} style={{ width: '100%' }}>
                    {availableInstances.find(i => i.id === adoptInstanceId)?.name || 'Select instance...'}
                  </MenuToggle>
                )}
              >
                {availableInstances.map(inst => (
                  <SelectOption key={inst.id} value={inst.id}>{inst.name}</SelectOption>
                ))}
              </Select>
            </FormGroup>
          )}
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSubmit}
          isDisabled={!childTypeName || (addChildMode === 'create' ? !newChildName.trim() : !adoptInstanceId)}>
          {addChildMode === 'create' ? 'Create' : 'Adopt'}
        </Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
