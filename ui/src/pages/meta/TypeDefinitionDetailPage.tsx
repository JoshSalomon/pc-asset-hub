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
  Label,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api } from '../../api/client'
import type { TypeDefinition, TypeDefinitionVersion, Role } from '../../types'
import { ConstraintsForm } from './TypeDefinitionListPage'

interface Props {
  role: Role
}

export default function TypeDefinitionDetailPage({ role }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const [typeDef, setTypeDef] = useState<TypeDefinition | null>(null)
  const [versions, setVersions] = useState<TypeDefinitionVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Edit description
  const [editingDesc, setEditingDesc] = useState(false)
  const [editDescValue, setEditDescValue] = useState('')
  const [editDescError, setEditDescError] = useState<string | null>(null)

  // Edit constraints (creates new version)
  const [editConstraintsOpen, setEditConstraintsOpen] = useState(false)
  const [editConstraints, setEditConstraints] = useState<Record<string, unknown>>({})
  const [editConstraintsError, setEditConstraintsError] = useState<string | null>(null)

  // Delete confirmation
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const loadTypeDef = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const [td, vers] = await Promise.all([
        api.typeDefinitions.get(id),
        api.typeDefinitions.listVersions(id),
      ])
      setTypeDef(td)
      setVersions(vers.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    loadTypeDef()
  }, [loadTypeDef])

  const handleUpdateDescription = async () => {
    if (!id) return
    setEditDescError(null)
    try {
      await api.typeDefinitions.update(id, { description: editDescValue })
      setEditingDesc(false)
      loadTypeDef()
    } catch (e) {
      setEditDescError(e instanceof Error ? e.message : 'Failed to update')
    }
  }

  const handleUpdateConstraints = async () => {
    if (!id) return
    setEditConstraintsError(null)
    try {
      // Clean up empty constraints
      const cleanConstraints: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(editConstraints)) {
        if (v !== undefined && v !== '' && v !== null) {
          cleanConstraints[k] = v
        }
      }
      await api.typeDefinitions.update(id, {
        constraints: Object.keys(cleanConstraints).length > 0 ? cleanConstraints : {},
      })
      setEditConstraintsOpen(false)
      loadTypeDef()
    } catch (e) {
      setEditConstraintsError(e instanceof Error ? e.message : 'Failed to update constraints')
    }
  }

  const handleDelete = async () => {
    if (!id) return
    setDeleteError(null)
    try {
      await api.typeDefinitions.delete(id)
      navigate('/schema/types')
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !typeDef) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!typeDef) return <PageSection><Alert variant="warning" title="Type definition not found" /></PageSection>

  const isReadOnly = typeDef.system

  // Get latest version constraints for the edit modal
  const latestVersion = versions.length > 0
    ? versions.reduce((a, b) => a.version_number > b.version_number ? a : b)
    : null

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/schema/types')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Types
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">
        {typeDef.name}
        {typeDef.system && <>{' '}<Label color="blue">System</Label></>}
      </Title>

      <DescriptionList style={{ marginTop: '1rem' }}>
        <DescriptionListGroup>
          <DescriptionListTerm>Name</DescriptionListTerm>
          <DescriptionListDescription>{typeDef.name}</DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Base Type</DescriptionListTerm>
          <DescriptionListDescription><Label>{typeDef.base_type}</Label></DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Description</DescriptionListTerm>
          <DescriptionListDescription>
            {editingDesc ? (
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                <TextInput
                  value={editDescValue}
                  onChange={(_e, v) => setEditDescValue(v)}
                  aria-label="Description"
                  style={{ width: '100%' }}
                />
                <Button variant="primary" size="sm" onClick={handleUpdateDescription}>Save</Button>
                <Button variant="link" size="sm" onClick={() => { setEditingDesc(false); setEditDescError(null) }}>Cancel</Button>
              </div>
            ) : (
              <>
                {typeDef.description || <span style={{ color: '#6a6e73' }}>No description</span>}
                {canEdit && !isReadOnly && (
                  <Button variant="link" size="sm" onClick={() => { setEditDescValue(typeDef.description || ''); setEditingDesc(true) }} style={{ marginLeft: '0.5rem' }} aria-label="Edit description">Edit</Button>
                )}
              </>
            )}
            {editDescError && <Alert variant="danger" title={editDescError} isInline style={{ marginTop: '0.5rem' }} />}
          </DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Latest Version</DescriptionListTerm>
          <DescriptionListDescription>V{typeDef.latest_version}</DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>ID</DescriptionListTerm>
          <DescriptionListDescription><code>{typeDef.id}</code></DescriptionListDescription>
        </DescriptionListGroup>
        <DescriptionListGroup>
          <DescriptionListTerm>Created</DescriptionListTerm>
          <DescriptionListDescription>{new Date(typeDef.created_at).toLocaleString()}</DescriptionListDescription>
        </DescriptionListGroup>
      </DescriptionList>

      {/* Current Constraints */}
      {latestVersion && Object.keys(latestVersion.constraints || {}).length > 0 && (
        <div style={{ marginTop: '1.5rem' }}>
          <Title headingLevel="h3">Current Constraints (V{latestVersion.version_number})</Title>
          <DescriptionList style={{ marginTop: '0.5rem' }}>
            {Object.entries(latestVersion.constraints).map(([key, value]) => (
              <DescriptionListGroup key={key}>
                <DescriptionListTerm>{key}</DescriptionListTerm>
                <DescriptionListDescription>
                  {Array.isArray(value) ? value.join(', ') : String(value)}
                </DescriptionListDescription>
              </DescriptionListGroup>
            ))}
          </DescriptionList>
        </div>
      )}

      {canEdit && !isReadOnly && (
        <Toolbar style={{ marginTop: '1rem' }}>
          <ToolbarContent>
            <ToolbarItem>
              <Button variant="secondary" onClick={() => {
                setEditConstraints(latestVersion?.constraints ? { ...latestVersion.constraints } : {})
                setEditConstraintsError(null)
                setEditConstraintsOpen(true)
              }}>Edit Constraints</Button>
            </ToolbarItem>
            <ToolbarItem>
              <Button variant="danger" onClick={() => setDeleteOpen(true)}>Delete</Button>
            </ToolbarItem>
          </ToolbarContent>
        </Toolbar>
      )}

      {/* Version History */}
      <Title headingLevel="h3" style={{ marginTop: '1.5rem' }}>Version History</Title>
      {versions.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No versions found.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Type definition versions">
          <Thead>
            <Tr>
              <Th>Version</Th>
              <Th>Constraints</Th>
              <Th>Created</Th>
            </Tr>
          </Thead>
          <Tbody>
            {versions.map((v) => (
              <Tr key={v.id}>
                <Td>V{v.version_number}</Td>
                <Td>
                  {Object.keys(v.constraints || {}).length > 0
                    ? JSON.stringify(v.constraints)
                    : <span style={{ color: '#6a6e73' }}>None</span>}
                </Td>
                <Td>{new Date(v.created_at).toLocaleString()}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {/* Edit Constraints Modal */}
      <Modal variant={ModalVariant.medium} isOpen={editConstraintsOpen} onClose={() => { setEditConstraintsOpen(false); setEditConstraintsError(null) }}>
        <ModalHeader title="Edit Constraints (creates new version)" />
        <ModalBody>
          {editConstraintsError && <Alert variant="danger" title={editConstraintsError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <ConstraintsForm
              baseType={typeDef.base_type}
              constraints={editConstraints}
              onChange={setEditConstraints}
              idPrefix="td-edit"
            />
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleUpdateConstraints}>Save</Button>
          <Button variant="link" onClick={() => { setEditConstraintsOpen(false); setEditConstraintsError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal variant={ModalVariant.small} isOpen={deleteOpen} onClose={() => { setDeleteOpen(false); setDeleteError(null) }}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete type definition <strong>{typeDef.name}</strong>?
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteOpen(false); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
