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
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api } from '../../api/client'
import type { Enum, Role } from '../../types'

interface Props {
  role: Role
}

export default function EnumListPage({ role }: Props) {
  const navigate = useNavigate()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const [enums, setEnums] = useState<Enum[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Create modal
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newValues, setNewValues] = useState('')
  const [createError, setCreateError] = useState<string | null>(null)

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<Enum | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const loadEnums = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await api.enums.list()
      setEnums(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load enums')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadEnums()
  }, [loadEnums])

  const handleCreate = async () => {
    if (!newName.trim()) return
    setCreateError(null)
    try {
      const values = newValues.trim()
        ? newValues.split(',').map((v) => v.trim()).filter(Boolean)
        : undefined
      await api.enums.create({ name: newName.trim(), values })
      setCreateOpen(false)
      setNewName('')
      setNewValues('')
      loadEnums()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleteError(null)
    try {
      await api.enums.delete(deleteTarget.id)
      setDeleteTarget(null)
      loadEnums()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  return (
    <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
      <Title headingLevel="h2">Enums</Title>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Toolbar>
        <ToolbarContent>
          {canEdit && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => setCreateOpen(true)}>Create Enum</Button>
            </ToolbarItem>
          )}
          <ToolbarItem>
            <Button variant="plain" onClick={loadEnums}>Refresh</Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {loading ? (
        <Spinner aria-label="Loading" />
      ) : enums.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No enums yet. Create one to get started.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Enums">
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>ID</Th>
              <Th>Created</Th>
              {canEdit && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {enums.map((en) => (
              <Tr key={en.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/schema/enums/${en.id}`)}>
                    {en.name}
                  </Button>
                </Td>
                <Td><code>{en.id.slice(0, 8)}...</code></Td>
                <Td>{new Date(en.created_at).toLocaleString()}</Td>
                {canEdit && (
                  <Td>
                    <Button variant="danger" size="sm" onClick={() => setDeleteTarget(en)}>Delete</Button>
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {/* Create Enum Modal */}
      <Modal variant={ModalVariant.small} isOpen={createOpen} onClose={() => { setCreateOpen(false); setCreateError(null) }}>
        <ModalHeader title="Create Enum" />
        <ModalBody>
          {createError && <Alert variant="danger" title={createError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="enum-name">
              <TextInput id="enum-name" value={newName} onChange={(_e, v) => setNewName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Initial Values (comma-separated)" fieldId="enum-values">
              <TextInput id="enum-values" value={newValues} onChange={(_e, v) => setNewValues(v)} placeholder="e.g. active, inactive, pending" />
            </FormGroup>
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
          Are you sure you want to delete enum <strong>{deleteTarget?.name}</strong>?
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteTarget(null); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
