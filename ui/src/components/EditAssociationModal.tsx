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
} from '@patternfly/react-core'

export interface AssociationEditData {
  name: string
  type: string
  sourceRole: string
  targetRole: string
  sourceCardinality: string
  targetCardinality: string
}

interface EditAssociationModalProps {
  isOpen: boolean
  onClose: () => void
  onSave: (data: AssociationEditData) => Promise<void>
  initialData: AssociationEditData & {
    sourceName?: string
    targetName?: string
  }
  showEntityTypeNames?: boolean
  allowTypeChange?: boolean
}

const STANDARD_OPTIONS = ['0..1', '0..n', '1', '1..n']
const CONTAINMENT_SOURCE_OPTIONS = ['0..1', '1']

function isStandard(value: string, options: string[]): boolean {
  return options.includes(value)
}

function parseCustom(value: string): { min: string; max: string } {
  const parts = value.split('..')
  return parts.length === 2 ? { min: parts[0], max: parts[1] } : { min: value, max: '' }
}

export default function EditAssociationModal({
  isOpen,
  onClose,
  onSave,
  initialData,
  showEntityTypeNames = false,
  allowTypeChange = false,
}: EditAssociationModalProps) {
  const [name, setName] = useState('')
  const [type, setType] = useState('')
  const [sourceRole, setSourceRole] = useState('')
  const [targetRole, setTargetRole] = useState('')
  const [sourceCard, setSourceCard] = useState('0..n')
  const [targetCard, setTargetCard] = useState('0..n')
  const [srcCustom, setSrcCustom] = useState(false)
  const [srcMin, setSrcMin] = useState('')
  const [srcMax, setSrcMax] = useState('')
  const [tgtCustom, setTgtCustom] = useState(false)
  const [tgtMin, setTgtMin] = useState('')
  const [tgtMax, setTgtMax] = useState('')
  const [error, setError] = useState<string | null>(null)

  // Reset form when modal opens with new data
  useEffect(() => {
    if (!isOpen) return
    setName(initialData.name)
    setType(initialData.type)
    setSourceRole(initialData.sourceRole)
    setTargetRole(initialData.targetRole)
    setError(null)

    // Source cardinality — detect custom
    const srcOptions = initialData.type === 'containment' ? CONTAINMENT_SOURCE_OPTIONS : STANDARD_OPTIONS
    if (isStandard(initialData.sourceCardinality, srcOptions)) {
      setSourceCard(initialData.sourceCardinality)
      setSrcCustom(false)
      setSrcMin('')
      setSrcMax('')
    } else {
      setSrcCustom(true)
      const parsed = parseCustom(initialData.sourceCardinality)
      setSrcMin(parsed.min)
      setSrcMax(parsed.max)
    }

    // Target cardinality — detect custom
    if (isStandard(initialData.targetCardinality, STANDARD_OPTIONS)) {
      setTargetCard(initialData.targetCardinality)
      setTgtCustom(false)
      setTgtMin('')
      setTgtMax('')
    } else {
      setTgtCustom(true)
      const parsed = parseCustom(initialData.targetCardinality)
      setTgtMin(parsed.min)
      setTgtMax(parsed.max)
    }
  }, [isOpen, initialData])

  // Reset source cardinality when type changes to containment
  useEffect(() => {
    if (type === 'containment') {
      setSrcCustom(false)
      if (!CONTAINMENT_SOURCE_OPTIONS.includes(sourceCard)) {
        setSourceCard('0..1')
      }
    }
  }, [type])

  const handleSave = async () => {
    setError(null)
    // Validate custom cardinality
    if (srcCustom && (!srcMin.trim() || !srcMax.trim())) {
      setError('Source cardinality: both min and max are required for custom values')
      return
    }
    if (tgtCustom && (!tgtMin.trim() || !tgtMax.trim())) {
      setError('Target cardinality: both min and max are required for custom values')
      return
    }
    const sc = srcCustom ? `${srcMin}..${srcMax}` : sourceCard
    const tc = tgtCustom ? `${tgtMin}..${tgtMax}` : targetCard
    try {
      await onSave({ name, type, sourceRole, targetRole, sourceCardinality: sc, targetCardinality: tc })
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save')
    }
  }

  const isContainment = type === 'containment'
  const srcOptions = isContainment ? CONTAINMENT_SOURCE_OPTIONS : STANDARD_OPTIONS

  return (
    <Modal variant={ModalVariant.small} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Edit Association" />
      <ModalBody>
        {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
        <Form>
          {showEntityTypeNames && (
            <>
              <FormGroup label="Source Entity Type" fieldId="edit-assoc-source-et">
                <TextInput id="edit-assoc-source-et" value={initialData.sourceName || ''} isDisabled />
              </FormGroup>
              <FormGroup label="Target Entity Type" fieldId="edit-assoc-target-et">
                <TextInput id="edit-assoc-target-et" value={initialData.targetName || ''} isDisabled />
              </FormGroup>
            </>
          )}
          <FormGroup label="Name" isRequired fieldId="edit-assoc-name">
            <TextInput id="edit-assoc-name" value={name} onChange={(_e, v) => setName(v)} isRequired />
          </FormGroup>
          {allowTypeChange ? (
            <FormGroup label="Type" fieldId="edit-assoc-type">
              <select id="edit-assoc-type" value={type}
                onChange={(e) => setType(e.target.value)} className="pf-v6-c-form-control">
                <option value="containment">containment</option>
                <option value="directional">directional</option>
                <option value="bidirectional">bidirectional</option>
              </select>
            </FormGroup>
          ) : (
            <FormGroup label="Type" fieldId="edit-assoc-type">
              <TextInput id="edit-assoc-type" value={type} isDisabled />
            </FormGroup>
          )}
          <FormGroup label="Source Role" fieldId="edit-assoc-source-role">
            <TextInput id="edit-assoc-source-role" value={sourceRole} onChange={(_e, v) => setSourceRole(v)} />
          </FormGroup>
          <FormGroup label="Target Role" fieldId="edit-assoc-target-role">
            <TextInput id="edit-assoc-target-role" value={targetRole} onChange={(_e, v) => setTargetRole(v)} />
          </FormGroup>
          <FormGroup label="Source Cardinality" fieldId="edit-assoc-source-card">
            {isContainment ? (
              <select id="edit-assoc-source-card"
                value={sourceCard}
                onChange={(e) => { setSrcCustom(false); setSourceCard(e.target.value) }}
                className="pf-v6-c-form-control">
                <option value="0..1">0..1</option>
                <option value="1">1</option>
              </select>
            ) : (
              <>
                <select id="edit-assoc-source-card"
                  value={srcCustom ? 'custom' : sourceCard}
                  onChange={(e) => {
                    if (e.target.value === 'custom') { setSrcCustom(true) }
                    else { setSrcCustom(false); setSourceCard(e.target.value) }
                  }}
                  className="pf-v6-c-form-control">
                  {srcOptions.map(o => <option key={o} value={o}>{o}</option>)}
                  <option value="custom">Custom</option>
                </select>
                {srcCustom && (
                  <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem', alignItems: 'center' }}>
                    <TextInput id="edit-assoc-src-min" value={srcMin}
                      onChange={(_e, v) => { if (v === '' || /^\d+$/.test(v)) setSrcMin(v) }}
                      placeholder="min" style={{ width: '5rem' }} />
                    <span>..</span>
                    <TextInput id="edit-assoc-src-max" value={srcMax}
                      onChange={(_e, v) => { if (v === '' || v === 'n' || /^\d+$/.test(v)) setSrcMax(v) }}
                      placeholder="max or n" style={{ width: '5rem' }} />
                  </div>
                )}
              </>
            )}
          </FormGroup>
          <FormGroup label="Target Cardinality" fieldId="edit-assoc-target-card">
            <select id="edit-assoc-target-card"
              value={tgtCustom ? 'custom' : targetCard}
              onChange={(e) => {
                if (e.target.value === 'custom') { setTgtCustom(true) }
                else { setTgtCustom(false); setTargetCard(e.target.value) }
              }}
              className="pf-v6-c-form-control">
              {STANDARD_OPTIONS.map(o => <option key={o} value={o}>{o}</option>)}
              <option value="custom">Custom</option>
            </select>
            {tgtCustom && (
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem', alignItems: 'center' }}>
                <TextInput id="edit-assoc-tgt-min" value={tgtMin}
                  onChange={(_e, v) => { if (v === '' || /^\d+$/.test(v)) setTgtMin(v) }}
                  placeholder="min" style={{ width: '5rem' }} />
                <span>..</span>
                <TextInput id="edit-assoc-tgt-max" value={tgtMax}
                  onChange={(_e, v) => { if (v === '' || v === 'n' || /^\d+$/.test(v)) setTgtMax(v) }}
                  placeholder="max or n" style={{ width: '5rem' }} />
              </div>
            )}
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button variant="primary" onClick={handleSave} isDisabled={!name.trim()}>Save</Button>
        <Button variant="link" onClick={onClose}>Cancel</Button>
      </ModalFooter>
    </Modal>
  )
}
