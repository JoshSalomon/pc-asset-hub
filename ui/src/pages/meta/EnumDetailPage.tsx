import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
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
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { ArrowUpIcon, ArrowDownIcon } from '@patternfly/react-icons'
import { api } from '../../api/client'
import type { Enum, EnumValue, Role } from '../../types'

interface Props {
  role: Role
}

export default function EnumDetailPage({ role }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const [enumData, setEnumData] = useState<Enum | null>(null)
  const [values, setValues] = useState<EnumValue[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Edit name
  const [editName, setEditName] = useState('')
  const [editNameOpen, setEditNameOpen] = useState(false)
  const [editNameError, setEditNameError] = useState<string | null>(null)

  // Add value
  const [addValueOpen, setAddValueOpen] = useState(false)
  const [newValue, setNewValue] = useState('')
  const [addValueError, setAddValueError] = useState<string | null>(null)

  // Delete confirmation
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const loadEnum = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const [en, vals] = await Promise.all([
        api.enums.get(id),
        api.enums.listValues(id),
      ])
      setEnumData(en)
      setValues(vals.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    loadEnum()
  }, [loadEnum])

  const handleUpdateName = async () => {
    if (!id || !editName.trim()) return
    setEditNameError(null)
    try {
      await api.enums.update(id, { name: editName.trim(), description: enumData?.description ?? '' })
      setEditNameOpen(false)
      loadEnum()
    } catch (e) {
      setEditNameError(e instanceof Error ? e.message : 'Failed to update')
    }
  }

  const handleAddValue = async () => {
    if (!id || !newValue.trim()) return
    setAddValueError(null)
    try {
      await api.enums.addValue(id, newValue.trim())
      setAddValueOpen(false)
      setNewValue('')
      loadEnum()
    } catch (e) {
      setAddValueError(e instanceof Error ? e.message : 'Failed to add value')
    }
  }

  const handleRemoveValue = async (valueId: string) => {
    if (!id) return
    try {
      await api.enums.removeValue(id, valueId)
      loadEnum()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove value')
    }
  }

  const handleReorderValue = async (index: number, direction: 'up' | 'down') => {
    if (!id) return
    const newVals = [...values]
    const swapIndex = direction === 'up' ? index - 1 : index + 1
    if (swapIndex < 0 || swapIndex >= newVals.length) return
    ;[newVals[index], newVals[swapIndex]] = [newVals[swapIndex], newVals[index]]
    try {
      await api.enums.reorderValues(id, newVals.map((v) => v.id))
      setValues(newVals)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to reorder')
    }
  }

  const handleDelete = async () => {
    if (!id) return
    setDeleteError(null)
    try {
      await api.enums.delete(id)
      navigate('/schema/enums')
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !enumData) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!enumData) return <PageSection><Alert variant="warning" title="Enum not found" /></PageSection>

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/schema/enums')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Enums
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">{enumData.name}</Title>

      <DescriptionList style={{ marginTop: '1rem' }}>
        <DescriptionListGroup>
          <DescriptionListTerm>Name</DescriptionListTerm>
          <DescriptionListDescription>
            {enumData.name}
            {canEdit && (
              <Button
                variant="link"
                size="sm"
                onClick={() => { setEditName(enumData.name); setEditNameOpen(true) }}
                style={{ marginLeft: '0.5rem' }}
                aria-label="Edit name"
              >
                Edit
              </Button>
            )}
          </DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Description</DescriptionListTerm>
          <DescriptionListDescription>
            {enumData.description || <span style={{ color: '#6a6e73' }}>No description</span>}
            {canEdit && (
              <Button variant="link" size="sm" onClick={async () => {
                const desc = window.prompt('Enter description:', enumData.description || '')
                if (desc !== null) {
                  try {
                    await api.enums.update(id!, { name: enumData.name, description: desc })
                    loadEnum()
                  } catch (e) {
                    setDeleteError(e instanceof Error ? e.message : 'Failed to update')
                  }
                }
              }} style={{ marginLeft: '0.5rem' }} aria-label="Edit description">Edit</Button>
            )}
          </DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>ID</DescriptionListTerm>
          <DescriptionListDescription><code>{enumData.id}</code></DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Created</DescriptionListTerm>
          <DescriptionListDescription>{new Date(enumData.created_at).toLocaleString()}</DescriptionListDescription>
        </DescriptionListGroup>
      </DescriptionList>

      {canEdit && (
        <Toolbar style={{ marginTop: '1rem' }}>
          <ToolbarContent>
            <ToolbarItem>
              <Button variant="danger" onClick={() => setDeleteOpen(true)}>Delete Enum</Button>
            </ToolbarItem>
          </ToolbarContent>
        </Toolbar>
      )}

      <Title headingLevel="h3" style={{ marginTop: '1.5rem' }}>Values</Title>

      {canEdit && (
        <Toolbar>
          <ToolbarContent>
            <ToolbarItem>
              <Button variant="primary" onClick={() => setAddValueOpen(true)}>Add Value</Button>
            </ToolbarItem>
          </ToolbarContent>
        </Toolbar>
      )}

      {values.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No values defined yet.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Enum values">
          <Thead>
            <Tr>
              <Th>Value</Th>
              <Th>Ordinal</Th>
              {canEdit && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {values.map((v, idx) => (
              <Tr key={v.id}>
                <Td>{v.value}</Td>
                <Td>{v.ordinal}</Td>
                {canEdit && (
                  <Td>
                    <Button
                      variant="plain"
                      size="sm"
                      onClick={() => handleReorderValue(idx, 'up')}
                      isDisabled={idx === 0}
                      aria-label="Move up"
                    >
                      <ArrowUpIcon />
                    </Button>
                    <Button
                      variant="plain"
                      size="sm"
                      onClick={() => handleReorderValue(idx, 'down')}
                      isDisabled={idx === values.length - 1}
                      aria-label="Move down"
                    >
                      <ArrowDownIcon />
                    </Button>
                    <Button variant="danger" size="sm" onClick={() => handleRemoveValue(v.id)}>Remove</Button>
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {/* Edit Name Modal */}
      <Modal variant={ModalVariant.small} isOpen={editNameOpen} onClose={() => { setEditNameOpen(false); setEditNameError(null) }}>
        <ModalHeader title="Edit Enum Name" />
        <ModalBody>
          {editNameError && <Alert variant="danger" title={editNameError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="edit-enum-name">
              <TextInput id="edit-enum-name" value={editName} onChange={(_e, v) => setEditName(v)} isRequired />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleUpdateName} isDisabled={!editName.trim()}>Save</Button>
          <Button variant="link" onClick={() => { setEditNameOpen(false); setEditNameError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Add Value Modal */}
      <Modal variant={ModalVariant.small} isOpen={addValueOpen} onClose={() => { setAddValueOpen(false); setAddValueError(null) }}>
        <ModalHeader title="Add Value" />
        <ModalBody>
          {addValueError && <Alert variant="danger" title={addValueError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Value" isRequired fieldId="enum-value">
              <TextInput id="enum-value" value={newValue} onChange={(_e, v) => setNewValue(v)} isRequired />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleAddValue} isDisabled={!newValue.trim()}>Add</Button>
          <Button variant="link" onClick={() => { setAddValueOpen(false); setAddValueError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal variant={ModalVariant.small} isOpen={deleteOpen} onClose={() => { setDeleteOpen(false); setDeleteError(null) }}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete enum <strong>{enumData.name}</strong>?
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteOpen(false); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
