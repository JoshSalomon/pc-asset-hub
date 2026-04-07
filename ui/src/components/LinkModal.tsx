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
}

export default function LinkModal({
  isOpen, onClose,
  catalogName, pins,
  schemaAssocs,
  onSubmit, error,
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
    const assoc = schemaAssocs.find(a => a.name === selectedAssocName && a.direction === 'outgoing')
    if (!assoc) return
    const targetPin = pins.find(p => p.entity_type_id === assoc.target_entity_type_id)
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
              {schemaAssocs.filter(a => a.type !== 'containment' && a.direction === 'outgoing').map(a => (
                <SelectOption key={a.name} value={a.name}>
                  {a.name} &rarr; {a.target_entity_type_name}
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
                  {linkTargetInstances.find(i => i.id === targetId)?.name || 'Select target instance...'}
                </MenuToggle>
              )}
            >
              {linkTargetInstances.map(inst => (
                <SelectOption key={inst.id} value={inst.id}>{inst.name}</SelectOption>
              ))}
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
