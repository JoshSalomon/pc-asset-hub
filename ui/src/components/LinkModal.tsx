import { useState, useEffect } from 'react'
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
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'
import type { EntityInstance, SnapshotAssociation, CatalogVersionPin } from '../types'
import { api } from '../api/client'

interface Props {
  isOpen: boolean
  onClose: () => void
  catalogName: string | undefined
  pins: CatalogVersionPin[]
  schemaAssocs: SnapshotAssociation[]
  onSubmit: (targetId: string, assocName: string) => Promise<void>
  error: string | null
  instanceNames?: Record<string, string>
}

export default function LinkModal({
  isOpen, onClose,
  catalogName, pins,
  schemaAssocs,
  onSubmit, error,
  instanceNames = {},
}: Props) {
  const [assocName, setAssocName] = useState('')
  const [targetId, setTargetId] = useState('')
  const [linkAssocSelectOpen, setLinkAssocSelectOpen] = useState(false)
  const [linkTargetSelectOpen, setLinkTargetSelectOpen] = useState(false)

  // Internal data state (previously managed by page)
  const [linkTargetInstances, setLinkTargetInstances] = useState<EntityInstance[]>([])

  // Load target instances when association selected
  const loadLinkTargetInstances = async (selectedAssocName: string) => {
    if (!catalogName) return
    const assoc = schemaAssocs.find(a => a.name === selectedAssocName &&
      (a.direction === 'outgoing' || (a.direction === 'incoming' && a.type === 'bidirectional')))
    if (!assoc) return
    const linkTypeId = assoc.direction === 'incoming' ? assoc.source_entity_type_id : assoc.target_entity_type_id
    const targetPin = pins.find(p => p.entity_type_id === linkTypeId)
    if (!targetPin) return
    try {
      const res = await api.instances.list(catalogName, targetPin.entity_type_name)
      setLinkTargetInstances(res.items || [])
    } catch { setLinkTargetInstances([]) }
  }

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setAssocName('')
      setTargetId('')
      setLinkTargetInstances([])
    }
  }, [isOpen])

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Link to Instance" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Association" isRequired fieldId="link-assoc">
            <Select
              id="link-assoc"
              isOpen={linkAssocSelectOpen}
              selected={assocName}
              onSelect={(_e, val) => {
                const v = val as string
                setAssocName(v)
                setLinkAssocSelectOpen(false)
                setTargetId('')
                loadLinkTargetInstances(v)
              }}
              onOpenChange={setLinkAssocSelectOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setLinkAssocSelectOpen(!linkAssocSelectOpen)} isExpanded={linkAssocSelectOpen} style={{ width: '100%' }}>
                  {assocName || 'Select association...'}
                </MenuToggle>
              )}
            >
              {schemaAssocs.filter(a => a.type !== 'containment' &&
              (a.direction === 'outgoing' || (a.direction === 'incoming' && a.type === 'bidirectional'))).map(a => (
                <SelectOption key={a.name} value={a.name} data-testid={`link-assoc-${a.name}`}>
                  {a.name} &rarr; {a.direction === 'incoming' ? a.source_entity_type_name : a.target_entity_type_name}
                </SelectOption>
              ))}
            </Select>
          </FormGroup>
          <FormGroup label="Target Instance" isRequired fieldId="link-target">
            <Select
              id="link-target"
              isOpen={linkTargetSelectOpen}
              selected={targetId}
              onSelect={(_e, val) => { setTargetId(val as string); setLinkTargetSelectOpen(false) }}
              onOpenChange={setLinkTargetSelectOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setLinkTargetSelectOpen(!linkTargetSelectOpen)} isExpanded={linkTargetSelectOpen} style={{ width: '100%' }}>
                  {(() => {
                    const inst = linkTargetInstances.find(i => i.id === targetId)
                    if (!inst) return 'Select target instance...'
                    const pName = inst.parent_instance_id ? instanceNames[inst.parent_instance_id] : undefined
                    return pName ? `${inst.name} (${pName})` : inst.name
                  })()}
                </MenuToggle>
              )}
            >
              {linkTargetInstances.map(inst => {
                const parentName = inst.parent_instance_id ? instanceNames[inst.parent_instance_id] : undefined
                const label = parentName ? `${inst.name} (${parentName})` : inst.name
                return <SelectOption key={inst.id} value={inst.id} data-testid={`link-target-${inst.id}`}>{label}</SelectOption>
              })}
            </Select>
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={() => onSubmit(targetId, assocName)} isDisabled={!assocName || !targetId}>Link</Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
