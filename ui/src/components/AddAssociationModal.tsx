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
import type { EntityType } from '../types'

export interface AddAssociationValues {
  name: string
  targetId: string
  type: string
  sourceRole: string
  targetRole: string
  sourceCardinality: string
  targetCardinality: string
  sourceCardCustom: boolean
  targetCardCustom: boolean
  sourceCardMin: string
  sourceCardMax: string
  targetCardMin: string
  targetCardMax: string
}

interface Props {
  isOpen: boolean
  onClose: () => void
  onSubmit: (values: AddAssociationValues) => Promise<void>
  entityTypes: EntityType[]
  currentEntityTypeId: string | undefined
  error: string | null
}

export default function AddAssociationModal({
  isOpen, onClose, onSubmit, entityTypes, currentEntityTypeId, error,
}: Props) {
  const [name, setName] = useState('')
  const [targetId, setTargetId] = useState('')
  const [targetOpen, setTargetOpen] = useState(false)
  const [type, setType] = useState('containment')
  const [typeOpen, setTypeOpen] = useState(false)
  const [sourceRole, setSourceRole] = useState('')
  const [targetRole, setTargetRole] = useState('')
  const [sourceCardinality, setSourceCardinality] = useState('0..n')
  const [targetCardinality, setTargetCardinality] = useState('0..n')
  const [sourceCardCustom, setSourceCardCustom] = useState(false)
  const [sourceCardMin, setSourceCardMin] = useState('')
  const [sourceCardMax, setSourceCardMax] = useState('')
  const [targetCardCustom, setTargetCardCustom] = useState(false)
  const [targetCardMin, setTargetCardMin] = useState('')
  const [targetCardMax, setTargetCardMax] = useState('')

  // Reset form when modal opens (handles the case where isOpen is toggled
  // without going through handleClose, e.g. after a successful submit)
  useEffect(() => {
    if (isOpen) {
      setName('')
      setTargetId('')
      setType('containment')
      setSourceRole('')
      setTargetRole('')
      setSourceCardinality('0..n')
      setTargetCardinality('0..n')
      setSourceCardCustom(false)
      setTargetCardCustom(false)
      setSourceCardMin('')
      setSourceCardMax('')
      setTargetCardMin('')
      setTargetCardMax('')
    }
  }, [isOpen])

  const handleClose = () => {
    setName('')
    setTargetId('')
    setType('containment')
    setSourceRole('')
    setTargetRole('')
    setSourceCardinality('0..n')
    setTargetCardinality('0..n')
    setSourceCardCustom(false)
    setTargetCardCustom(false)
    setSourceCardMin('')
    setSourceCardMax('')
    setTargetCardMin('')
    setTargetCardMax('')
    onClose()
  }

  const handleSubmit = async () => {
    await onSubmit({
      name, targetId, type, sourceRole, targetRole,
      sourceCardinality, targetCardinality,
      sourceCardCustom, targetCardCustom,
      sourceCardMin, sourceCardMax, targetCardMin, targetCardMax,
    })
  }

  return (
    <Modal variant={ModalVariant.small} isOpen={isOpen} onClose={handleClose}>
      <ModalHeader title="Add Association" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          <FormGroup label="Name" isRequired fieldId="assoc-name">
            <TextInput id="assoc-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
          </FormGroup>
          <FormGroup label="Target Entity Type" isRequired fieldId="assoc-target">
            <Select
              isOpen={targetOpen}
              selected={targetId}
              onSelect={(_e, value) => { setTargetId(value as string); setTargetOpen(false) }}
              onOpenChange={setTargetOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setTargetOpen(!targetOpen)} isExpanded={targetOpen}>
                  {entityTypes.find((et) => et.id === targetId)?.name || 'Select target'}
                </MenuToggle>
              )}
            >
              {entityTypes.filter((et) => et.id !== currentEntityTypeId).map((et) => (
                <SelectOption key={et.id} value={et.id}>{et.name}</SelectOption>
              ))}
            </Select>
          </FormGroup>
          <FormGroup label="Type" isRequired fieldId="assoc-type">
            <Select
              isOpen={typeOpen}
              selected={type}
              onSelect={(_e, value) => {
                const newType = value as string
                setType(newType)
                setTypeOpen(false)
                if (newType === 'containment') {
                  setSourceCardCustom(false)
                  if (sourceCardinality !== '1' && sourceCardinality !== '0..1') {
                    setSourceCardinality('0..1')
                  }
                }
              }}
              onOpenChange={setTypeOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setTypeOpen(!typeOpen)} isExpanded={typeOpen}>{type}</MenuToggle>
              )}
            >
              <SelectOption value="containment">containment</SelectOption>
              <SelectOption value="directional">directional</SelectOption>
              <SelectOption value="bidirectional">bidirectional</SelectOption>
            </Select>
          </FormGroup>
          <FormGroup label="Source Role" fieldId="assoc-source-role">
            <TextInput id="assoc-source-role" value={sourceRole} onChange={(_e, v) => setSourceRole(v)} />
          </FormGroup>
          <FormGroup label="Target Role" fieldId="assoc-target-role">
            <TextInput id="assoc-target-role" value={targetRole} onChange={(_e, v) => setTargetRole(v)} />
          </FormGroup>
          <FormGroup label="Source Cardinality" fieldId="assoc-source-cardinality">
            {type === 'containment' ? (
              <select
                id="assoc-source-cardinality"
                value={sourceCardinality === '1' ? '1' : '0..1'}
                onChange={(e) => { setSourceCardCustom(false); setSourceCardinality(e.target.value) }}
                className="pf-v6-c-form-control"
              >
                <option value="0..1">0..1</option>
                <option value="1">1</option>
              </select>
            ) : (
              <>
                <select
                  id="assoc-source-cardinality"
                  value={sourceCardCustom ? 'custom' : sourceCardinality}
                  onChange={(e) => {
                    if (e.target.value === 'custom') {
                      setSourceCardCustom(true)
                    } else {
                      setSourceCardCustom(false)
                      setSourceCardinality(e.target.value)
                    }
                  }}
                  className="pf-v6-c-form-control"
                >
                  <option value="0..1">0..1</option>
                  <option value="0..n">0..n</option>
                  <option value="1">1</option>
                  <option value="1..n">1..n</option>
                  <option value="custom">Custom</option>
                </select>
                {sourceCardCustom && (
                  <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem', alignItems: 'center' }}>
                    <TextInput id="assoc-source-card-min" value={sourceCardMin} onChange={(_e, v) => { if (v === '' || /^\d+$/.test(v)) setSourceCardMin(v) }} placeholder="min" style={{ width: '5rem' }} />
                    <span>..</span>
                    <TextInput id="assoc-source-card-max" value={sourceCardMax} onChange={(_e, v) => { if (v === '' || v === 'n' || /^\d+$/.test(v)) setSourceCardMax(v) }} placeholder="max or n" style={{ width: '5rem' }} />
                  </div>
                )}
              </>
            )}
          </FormGroup>
          <FormGroup label="Target Cardinality" fieldId="assoc-target-cardinality">
            <select
              id="assoc-target-cardinality"
              value={targetCardCustom ? 'custom' : targetCardinality}
              onChange={(e) => {
                if (e.target.value === 'custom') {
                  setTargetCardCustom(true)
                } else {
                  setTargetCardCustom(false)
                  setTargetCardinality(e.target.value)
                }
              }}
              className="pf-v6-c-form-control"
            >
              <option value="0..1">0..1</option>
              <option value="0..n">0..n</option>
              <option value="1">1</option>
              <option value="1..n">1..n</option>
              <option value="custom">Custom</option>
            </select>
            {targetCardCustom && (
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem', alignItems: 'center' }}>
                <TextInput id="assoc-target-card-min" value={targetCardMin} onChange={(_e, v) => { if (v === '' || /^\d+$/.test(v)) setTargetCardMin(v) }} placeholder="min" style={{ width: '5rem' }} />
                <span>..</span>
                <TextInput id="assoc-target-card-max" value={targetCardMax} onChange={(_e, v) => { if (v === '' || v === 'n' || /^\d+$/.test(v)) setTargetCardMax(v) }} placeholder="max or n" style={{ width: '5rem' }} />
              </div>
            )}
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSubmit} isDisabled={!targetId || !name.trim()}>Add</Button>
        <Button variant="link" onClick={handleClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
