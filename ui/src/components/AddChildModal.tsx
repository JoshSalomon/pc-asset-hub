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
import type { EntityInstance, SnapshotAssociation, SnapshotAttribute, CatalogVersionPin } from '../types'
import { api } from '../api/client'
import { buildTypedAttrs } from '../utils/buildTypedAttrs'
import AttributeFormFields from './AttributeFormFields'

export interface AddChildCreateData {
  name: string
  description: string
  attrs: Record<string, unknown>
}

export interface AddChildAdoptData {
  adoptInstanceId: string
}

interface Props {
  isOpen: boolean
  onClose: () => void
  catalogName: string | undefined
  pins: CatalogVersionPin[]
  schemaAssocs: SnapshotAssociation[]
  onSubmit: (childType: string, mode: 'create' | 'adopt', data: AddChildCreateData | AddChildAdoptData) => Promise<void>
  error: string | null
  initialChildType?: string
}

export default function AddChildModal({
  isOpen, onClose,
  catalogName, pins,
  schemaAssocs,
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

  // Internal data state (previously managed by page)
  const [childSchemaAttrs, setChildSchemaAttrs] = useState<SnapshotAttribute[]>([])
  const [childEnumValues, setChildEnumValues] = useState<Record<string, string[]>>({})
  const [availableInstances, setAvailableInstances] = useState<EntityInstance[]>([])

  // Load available instances for adopt mode
  const loadAvailableInstances = async (typeName: string) => {
    if (!catalogName || !typeName) { setAvailableInstances([]); return }
    try {
      const res = await api.instances.list(catalogName, typeName)
      setAvailableInstances((res.items || []).filter((i: EntityInstance) => !i.parent_instance_id))
    } catch { setAvailableInstances([]) }
  }

  // Load child type schema attributes and enum values
  const loadChildSchema = async (typeName: string) => {
    if (!typeName || !pins.length) { setChildSchemaAttrs([]); return }
    const pin = pins.find(p => p.entity_type_name === typeName)
    if (!pin) { setChildSchemaAttrs([]); return }
    try {
      const snapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
      setChildSchemaAttrs(snapshot.attributes || [])
      const cache: Record<string, string[]> = {}
      for (const attr of snapshot.attributes || []) {
        if (attr.base_type === 'enum' && attr.type_definition_version_id) {
          const constraintValues = (attr.constraints?.values as string[]) || []
          if (constraintValues.length > 0) {
            cache[attr.type_definition_version_id] = constraintValues
          }
        }
      }
      setChildEnumValues(cache)
    } catch { setChildSchemaAttrs([]) }
  }

  const handleChildTypeChange = (typeName: string) => {
    loadAvailableInstances(typeName)
    loadChildSchema(typeName)
  }

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setChildTypeName(initialChildType || '')
      setAddChildMode('create')
      setNewChildName('')
      setNewChildDesc('')
      setNewChildAttrs({})
      setAdoptInstanceId('')
      setChildSchemaAttrs([])
      setChildEnumValues({})
      setAvailableInstances([])
      if (initialChildType) {
        handleChildTypeChange(initialChildType)
      }
    }
  }, [isOpen, initialChildType])

  const handleSubmit = async () => {
    if (!childTypeName) return
    if (addChildMode === 'create') {
      const typedAttrs = buildTypedAttrs(newChildAttrs, childSchemaAttrs)
      await onSubmit(childTypeName, 'create', {
        name: newChildName,
        description: newChildDesc,
        attrs: typedAttrs,
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
                handleChildTypeChange(v)
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
