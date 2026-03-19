import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Toolbar,
  ToolbarItem,
  ToolbarContent,
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
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
  Label,
  EmptyState,
  EmptyStateBody,
  Spinner,
  HelperText,
  HelperTextItem,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, CatalogVersion, Role } from '../../types'
import { statusColor } from '../../utils/statusColor'

const DNS_LABEL_REGEX = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/

function validateCatalogName(name: string): string | null {
  if (!name) return 'Name is required'
  if (name.length > 63) return 'Name must be at most 63 characters'
  if (!DNS_LABEL_REGEX.test(name)) return 'Must be lowercase alphanumeric and hyphens, starting and ending with alphanumeric'
  return null
}

export default function CatalogListPage({ role }: { role: Role }) {
  const navigate = useNavigate()
  const [catalogs, setCatalogs] = useState<Catalog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Create modal
  const [createOpen, setCreateOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [newCvId, setNewCvId] = useState('')
  const [cvSelectOpen, setCvSelectOpen] = useState(false)
  const [availableCVs, setAvailableCVs] = useState<CatalogVersion[]>([])
  const [createError, setCreateError] = useState<string | null>(null)
  const [nameError, setNameError] = useState<string | null>(null)

  // Delete modal
  const [deleteTarget, setDeleteTarget] = useState<Catalog | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const canCreate = role === 'RW' || role === 'Admin' || role === 'SuperAdmin'

  const loadCatalogs = useCallback(async () => {
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const res = await api.catalogs.list()
      setCatalogs(res.items || [])
      setTotal(res.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalogs')
    } finally {
      setLoading(false)
    }
  }, [role])

  useEffect(() => {
    loadCatalogs()
  }, [loadCatalogs])

  const loadCVs = async () => {
    try {
      const res = await api.catalogVersions.list()
      setAvailableCVs(res.items || [])
    } catch { /* ignore */ }
  }

  const handleCreate = async () => {
    const validationError = validateCatalogName(newName)
    if (validationError) {
      setNameError(validationError)
      return
    }
    setCreateError(null)
    try {
      await api.catalogs.create({
        name: newName,
        description: newDesc || undefined,
        catalog_version_id: newCvId,
      })
      setCreateOpen(false)
      setNewName('')
      setNewDesc('')
      setNewCvId('')
      loadCatalogs()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    setDeleteError(null)
    try {
      await api.catalogs.delete(deleteTarget.name)
      setDeleteTarget(null)
      loadCatalogs()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  return (
    <PageSection padding={{ default: 'noPadding' }}>
      <Title headingLevel="h2" style={{ marginTop: '1rem' }}>Catalogs</Title>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Toolbar>
        <ToolbarContent>
          {canCreate && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => { setCreateOpen(true); loadCVs() }}>Create Catalog</Button>
            </ToolbarItem>
          )}
          <ToolbarItem>
            <Button variant="plain" onClick={loadCatalogs}>Refresh</Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {loading ? (
        <Spinner aria-label="Loading" />
      ) : catalogs.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No catalogs yet. Create one to get started.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Catalogs">
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>Description</Th>
              <Th>Catalog Version</Th>
              <Th>Status</Th>
              <Th>Created</Th>
              {canCreate && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {catalogs.map((cat) => (
              <Tr key={cat.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/catalogs/${cat.name}`)}>
                    {cat.name}
                  </Button>
                </Td>
                <Td style={{ maxWidth: '30rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {cat.description || ''}
                </Td>
                <Td>{cat.catalog_version_label || cat.catalog_version_id.slice(0, 8) + '...'}</Td>
                <Td>
                  <Label color={statusColor(cat.validation_status)}>{cat.validation_status}</Label>
                  {cat.published && <Label color="purple" style={{ marginLeft: '0.25rem' }}>published</Label>}
                </Td>
                <Td>{new Date(cat.created_at).toLocaleString()}</Td>
                {canCreate && (
                  <Td>
                    <Button variant="danger" size="sm" onClick={() => { setDeleteTarget(cat); setDeleteError(null) }}>Delete</Button>
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
      <p style={{ marginTop: '0.5rem' }}>Total: {total}</p>

      {/* Create Catalog Modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={createOpen}
        onClose={() => { setCreateOpen(false); setCreateError(null); setNameError(null) }}
      >
        <ModalHeader title="Create Catalog" />
        <ModalBody>
          {createError && <Alert variant="danger" title={createError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="cat-name">
              <TextInput
                id="cat-name"
                value={newName}
                onChange={(_e, v) => { setNewName(v); setNameError(validateCatalogName(v)) }}
                isRequired
                placeholder="e.g. production-app-a"
                validated={nameError ? 'error' : 'default'}
              />
              {nameError && (
                <HelperText>
                  <HelperTextItem variant="error">{nameError}</HelperTextItem>
                </HelperText>
              )}
            </FormGroup>
            <FormGroup label="Description" fieldId="cat-desc">
              <TextInput id="cat-desc" value={newDesc} onChange={(_e, v) => setNewDesc(v)} />
            </FormGroup>
            <FormGroup label="Catalog Version" isRequired fieldId="cat-cv">
              <Select
                isOpen={cvSelectOpen}
                selected={newCvId}
                onSelect={(_e, value) => { setNewCvId(value as string); setCvSelectOpen(false) }}
                onOpenChange={setCvSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setCvSelectOpen(!cvSelectOpen)} isExpanded={cvSelectOpen} style={{ width: '100%' }}>
                    {newCvId ? availableCVs.find(cv => cv.id === newCvId)?.version_label || newCvId : 'Select a catalog version'}
                  </MenuToggle>
                )}
              >
                {availableCVs.map(cv => (
                  <SelectOption key={cv.id} value={cv.id}>{cv.version_label}</SelectOption>
                ))}
              </Select>
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreate} isDisabled={!newName || !!nameError || !newCvId}>Create</Button>
          <Button variant="link" onClick={() => { setCreateOpen(false); setCreateError(null); setNameError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={deleteTarget !== null}
        onClose={() => { setDeleteTarget(null); setDeleteError(null) }}
      >
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete catalog <strong>{deleteTarget?.name}</strong>? All entity instances in this catalog will be deleted. This action cannot be undone.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteTarget(null); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
