import { useEffect, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Button,
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Form,
  FormGroup,
  TextInput,
  Alert,
  Spinner,
  EmptyState,
  EmptyStateBody,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  Label,
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
  NumberInput,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api } from '../../api/client'
import type { TypeDefinition, BaseType, Role } from '../../types'

const BASE_TYPES: BaseType[] = ['string', 'integer', 'number', 'boolean', 'date', 'url', 'enum', 'list', 'json']

interface Props {
  role: Role
}

function ConstraintsForm({
  baseType,
  constraints,
  onChange,
  idPrefix,
}: {
  baseType: BaseType
  constraints: Record<string, unknown>
  onChange: (c: Record<string, unknown>) => void
  idPrefix: string
}) {
  const [newEnumValue, setNewEnumValue] = useState('')
  const [elementTypeOpen, setElementTypeOpen] = useState(false)

  switch (baseType) {
    case 'string':
      return (
        <>
          <FormGroup label="Max Length" fieldId={`${idPrefix}-maxlen`}>
            <NumberInput
              id={`${idPrefix}-maxlen`}
              value={(constraints.max_length as number) || undefined}
              min={0}
              onMinus={() => onChange({ ...constraints, max_length: Math.max(0, ((constraints.max_length as number) || 1) - 1) })}
              onPlus={() => onChange({ ...constraints, max_length: ((constraints.max_length as number) || 0) + 1 })}
              onChange={(e) => {
                const v = parseInt((e.target as HTMLInputElement).value, 10)
                onChange({ ...constraints, max_length: isNaN(v) ? undefined : v })
              }}
            />
          </FormGroup>
          <FormGroup label="Pattern (regex)" fieldId={`${idPrefix}-pattern`}>
            <TextInput
              id={`${idPrefix}-pattern`}
              value={(constraints.pattern as string) || ''}
              onChange={(_e, v) => onChange({ ...constraints, pattern: v || undefined })}
              placeholder="e.g. ^[a-z]+$"
            />
          </FormGroup>
          <FormGroup fieldId={`${idPrefix}-multiline`}>
            <label>
              <input
                type="checkbox"
                id={`${idPrefix}-multiline`}
                checked={!!constraints.multiline}
                onChange={(e) => onChange({ ...constraints, multiline: e.target.checked || undefined })}
              />
              {' '}Multiline
            </label>
          </FormGroup>
        </>
      )
    case 'integer':
    case 'number':
      return (
        <>
          <FormGroup label="Min" fieldId={`${idPrefix}-min`}>
            <TextInput
              id={`${idPrefix}-min`}
              type="number"
              value={constraints.min != null ? String(constraints.min) : ''}
              onChange={(_e, v) => {
                const parsed = baseType === 'integer' ? parseInt(v, 10) : parseFloat(v)
                onChange({ ...constraints, min: v === '' ? undefined : (isNaN(parsed) ? undefined : parsed) })
              }}
            />
          </FormGroup>
          <FormGroup label="Max" fieldId={`${idPrefix}-max`}>
            <TextInput
              id={`${idPrefix}-max`}
              type="number"
              value={constraints.max != null ? String(constraints.max) : ''}
              onChange={(_e, v) => {
                const parsed = baseType === 'integer' ? parseInt(v, 10) : parseFloat(v)
                onChange({ ...constraints, max: v === '' ? undefined : (isNaN(parsed) ? undefined : parsed) })
              }}
            />
          </FormGroup>
        </>
      )
    case 'enum': {
      const values = (constraints.values as string[]) || []
      return (
        <>
          <FormGroup label="Values" fieldId={`${idPrefix}-values`}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.25rem', marginBottom: '0.5rem' }}>
              {values.map((v, i) => (
                <Label key={i} onClose={() => {
                  const next = values.filter((_, idx) => idx !== i)
                  onChange({ ...constraints, values: next })
                }}>
                  {v}
                </Label>
              ))}
            </div>
            <div style={{ display: 'flex', gap: '0.5rem' }}>
              <TextInput
                id={`${idPrefix}-new-value`}
                value={newEnumValue}
                onChange={(_e, v) => setNewEnumValue(v)}
                placeholder="New value"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && newEnumValue.trim()) {
                    e.preventDefault()
                    onChange({ ...constraints, values: [...values, newEnumValue.trim()] })
                    setNewEnumValue('')
                  }
                }}
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  if (newEnumValue.trim()) {
                    onChange({ ...constraints, values: [...values, newEnumValue.trim()] })
                    setNewEnumValue('')
                  }
                }}
                isDisabled={!newEnumValue.trim()}
              >
                Add
              </Button>
            </div>
          </FormGroup>
        </>
      )
    }
    case 'list':
      return (
        <>
          <FormGroup label="Element Base Type" fieldId={`${idPrefix}-elem-type`}>
            <Select
              isOpen={elementTypeOpen}
              selected={(constraints.element_base_type as string) || ''}
              onSelect={(_e, value) => {
                onChange({ ...constraints, element_base_type: value as string })
                setElementTypeOpen(false)
              }}
              onOpenChange={setElementTypeOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setElementTypeOpen(!elementTypeOpen)} isExpanded={elementTypeOpen}>
                  {(constraints.element_base_type as string) || 'Select...'}
                </MenuToggle>
              )}
            >
              {['string', 'integer', 'number', 'boolean'].map(t => (
                <SelectOption key={t} value={t}>{t}</SelectOption>
              ))}
            </Select>
          </FormGroup>
          <FormGroup label="Max Length" fieldId={`${idPrefix}-list-maxlen`}>
            <NumberInput
              id={`${idPrefix}-list-maxlen`}
              value={(constraints.max_length as number) || undefined}
              min={0}
              onMinus={() => onChange({ ...constraints, max_length: Math.max(0, ((constraints.max_length as number) || 1) - 1) })}
              onPlus={() => onChange({ ...constraints, max_length: ((constraints.max_length as number) || 0) + 1 })}
              onChange={(e) => {
                const v = parseInt((e.target as HTMLInputElement).value, 10)
                onChange({ ...constraints, max_length: isNaN(v) ? undefined : v })
              }}
            />
          </FormGroup>
        </>
      )
    case 'boolean':
    case 'date':
    case 'url':
    case 'json':
      return null
    default:
      return null
  }
}

export { ConstraintsForm }

function baseTypeColor(bt: string): 'blue' | 'green' | 'purple' | 'orange' | 'grey' {
  switch (bt) {
    case 'string': return 'blue'
    case 'integer': case 'number': return 'green'
    case 'enum': return 'purple'
    case 'boolean': return 'orange'
    default: return 'grey'
  }
}

export default function TypeDefinitionListPage({ role }: Props) {
  const navigate = useNavigate()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const [typeDefs, setTypeDefs] = useState<TypeDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create modal
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newBaseType, setNewBaseType] = useState<BaseType>('string')
  const [newBaseTypeOpen, setNewBaseTypeOpen] = useState(false)
  const [newConstraints, setNewConstraints] = useState<Record<string, unknown>>({})
  const [createError, setCreateError] = useState<string | null>(null)

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<TypeDefinition | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const loadTypeDefs = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.typeDefinitions.list()
      setTypeDefs(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load type definitions')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadTypeDefs()
  }, [loadTypeDefs])

  const handleCreate = async () => {
    if (!newName.trim()) return
    setCreateError(null)
    try {
      // Clean up empty constraints
      const cleanConstraints: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(newConstraints)) {
        if (v !== undefined && v !== '' && v !== null) {
          cleanConstraints[k] = v
        }
      }
      await api.typeDefinitions.create({
        name: newName.trim(),
        description: newDesc.trim() || undefined,
        base_type: newBaseType,
        constraints: Object.keys(cleanConstraints).length > 0 ? cleanConstraints : undefined,
      })
      setCreateOpen(false)
      setNewName('')
      setNewDesc('')
      setNewBaseType('string')
      setNewConstraints({})
      loadTypeDefs()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleteError(null)
    try {
      await api.typeDefinitions.delete(deleteTarget.id)
      setDeleteTarget(null)
      loadTypeDefs()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  return (
    <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
      <Title headingLevel="h2">Type Definitions</Title>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Toolbar>
        <ToolbarContent>
          {canEdit && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => setCreateOpen(true)}>Create Type Definition</Button>
            </ToolbarItem>
          )}
          <ToolbarItem>
            <Button variant="plain" onClick={loadTypeDefs}>Refresh</Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {loading ? (
        <Spinner aria-label="Loading" />
      ) : typeDefs.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No type definitions yet. Create one to get started.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Type definitions">
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>Base Type</Th>
              <Th>Latest Version</Th>
              <Th>Description</Th>
              <Th>Created</Th>
              {canEdit && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {typeDefs.map((td) => (
              <Tr key={td.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/schema/types/${td.id}`)}>
                    {td.name}
                  </Button>
                  {td.system && <>{' '}<Label color="blue" isCompact>System</Label></>}
                </Td>
                <Td><Label color={baseTypeColor(td.base_type)}>{td.base_type}</Label></Td>
                <Td>V{td.latest_version}</Td>
                <Td style={{ maxWidth: '20rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{td.description || '-'}</Td>
                <Td>{new Date(td.created_at).toLocaleString()}</Td>
                {canEdit && (
                  <Td>
                    {!td.system && (
                      <Button variant="danger" size="sm" onClick={() => setDeleteTarget(td)}>Delete</Button>
                    )}
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {/* Create Type Definition Modal */}
      <Modal variant={ModalVariant.medium} isOpen={createOpen} onClose={() => { setCreateOpen(false); setCreateError(null) }}>
        <ModalHeader title="Create Type Definition" />
        <ModalBody>
          {createError && <Alert variant="danger" title={createError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="td-name">
              <TextInput id="td-name" value={newName} onChange={(_e, v) => setNewName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="td-desc">
              <TextInput id="td-desc" value={newDesc} onChange={(_e, v) => setNewDesc(v)} placeholder="Optional description" />
            </FormGroup>
            <FormGroup label="Base Type" isRequired fieldId="td-base-type">
              <Select
                isOpen={newBaseTypeOpen}
                selected={newBaseType}
                onSelect={(_e, value) => {
                  setNewBaseType(value as BaseType)
                  setNewBaseTypeOpen(false)
                  setNewConstraints({})
                }}
                onOpenChange={setNewBaseTypeOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setNewBaseTypeOpen(!newBaseTypeOpen)} isExpanded={newBaseTypeOpen}>
                    {newBaseType}
                  </MenuToggle>
                )}
              >
                {BASE_TYPES.map(bt => (
                  <SelectOption key={bt} value={bt}>{bt}</SelectOption>
                ))}
              </Select>
            </FormGroup>
            <ConstraintsForm
              baseType={newBaseType}
              constraints={newConstraints}
              onChange={setNewConstraints}
              idPrefix="td-create"
            />
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreate} isDisabled={!newName.trim()}>Create</Button>
          <Button variant="link" onClick={() => { setCreateOpen(false); setCreateError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal variant={ModalVariant.small} isOpen={deleteTarget !== null} onClose={() => { setDeleteTarget(null); setDeleteError(null) }}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete type definition <strong>{deleteTarget?.name}</strong>?
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteTarget(null); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
