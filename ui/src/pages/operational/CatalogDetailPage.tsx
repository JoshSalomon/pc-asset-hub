import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Tabs,
  Tab,
  TabTitleText,
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
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, CatalogVersionPin, EntityInstance, SnapshotAttribute, Role } from '../../types'

export default function CatalogDetailPage({ role }: { role: Role }) {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()

  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pins, setPins] = useState<CatalogVersionPin[]>([])
  const [activeTab, setActiveTab] = useState<string>('')

  // Instance state per entity type
  const [instances, setInstances] = useState<EntityInstance[]>([])
  const [instTotal, setInstTotal] = useState(0)
  const [instLoading, setInstLoading] = useState(false)

  // Schema attributes for the active entity type
  const [schemaAttrs, setSchemaAttrs] = useState<SnapshotAttribute[]>([])

  // Create instance modal
  const [createOpen, setCreateOpen] = useState(false)
  const [newInstName, setNewInstName] = useState('')
  const [newInstDesc, setNewInstDesc] = useState('')
  const [newInstAttrs, setNewInstAttrs] = useState<Record<string, string>>({})
  const [createError, setCreateError] = useState<string | null>(null)

  // Edit instance modal
  const [editTarget, setEditTarget] = useState<EntityInstance | null>(null)
  const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState('')
  const [editAttrs, setEditAttrs] = useState<Record<string, string>>({})
  const [editError, setEditError] = useState<string | null>(null)

  // Delete instance modal
  const [deleteTarget, setDeleteTarget] = useState<EntityInstance | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Enum values cache for enum dropdowns
  const [enumValues, setEnumValues] = useState<Record<string, string[]>>({})

  const canWrite = role === 'RW' || role === 'Admin' || role === 'SuperAdmin'

  const loadCatalog = useCallback(async () => {
    if (!name) return
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const cat = await api.catalogs.get(name)
      setCatalog(cat)
      // Load pins from the CV
      const pinsRes = await api.catalogVersions.listPins(cat.catalog_version_id)
      setPins(pinsRes.items || [])
      if (pinsRes.items?.length && !activeTab) {
        setActiveTab(pinsRes.items[0].entity_type_name)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalog')
    } finally {
      setLoading(false)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [name, role])

  useEffect(() => { loadCatalog() }, [loadCatalog])

  const loadInstances = useCallback(async () => {
    if (!name || !activeTab) return
    setAuthRole(role)
    setInstLoading(true)
    try {
      const res = await api.instances.list(name, activeTab)
      setInstances(res.items || [])
      setInstTotal(res.total)
    } catch {
      setInstances([])
      setInstTotal(0)
    } finally {
      setInstLoading(false)
    }
  }, [name, activeTab, role])

  const loadSchema = useCallback(async () => {
    if (!activeTab || !pins.length) return
    const pin = pins.find(p => p.entity_type_name === activeTab)
    if (!pin) return
    try {
      const snapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
      setSchemaAttrs(snapshot.attributes || [])
      // Load enum values for enum attributes
      const enumCache: Record<string, string[]> = {}
      for (const attr of snapshot.attributes || []) {
        if (attr.type === 'enum' && attr.enum_id && !enumCache[attr.enum_id]) {
          try {
            const res = await api.enums.listValues(attr.enum_id)
            enumCache[attr.enum_id] = (res.items || []).map(v => v.value)
          } catch { /* ignore */ }
        }
      }
      setEnumValues(enumCache)
    } catch { /* ignore */ }
  }, [activeTab, pins])

  useEffect(() => { loadInstances() }, [loadInstances])
  useEffect(() => { loadSchema() }, [loadSchema])

  const handleCreate = async () => {
    if (!name || !activeTab || !newInstName.trim()) return
    setCreateError(null)
    try {
      const attrs: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(newInstAttrs)) {
        if (v !== '') {
          const schemaAttr = schemaAttrs.find(a => a.name === k)
          if (schemaAttr?.type === 'number') {
            attrs[k] = parseFloat(v)
          } else {
            attrs[k] = v
          }
        }
      }
      await api.instances.create(name, activeTab, {
        name: newInstName.trim(),
        description: newInstDesc.trim() || undefined,
        attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
      })
      setCreateOpen(false)
      setNewInstName('')
      setNewInstDesc('')
      setNewInstAttrs({})
      await loadInstances()
    } catch (e) {
      setCreateError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const openEdit = (inst: EntityInstance) => {
    setEditTarget(inst)
    setEditName(inst.name)
    setEditDesc(inst.description)
    const attrs: Record<string, string> = {}
    for (const av of inst.attributes || []) {
      attrs[av.name] = av.value != null ? String(av.value) : ''
    }
    setEditAttrs(attrs)
    setEditError(null)
  }

  const handleEdit = async () => {
    if (!name || !activeTab || !editTarget) return
    setEditError(null)
    try {
      const attrs: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(editAttrs)) {
        const schemaAttr = schemaAttrs.find(a => a.name === k)
        if (schemaAttr?.type === 'number' && v !== '') {
          attrs[k] = parseFloat(v)
        } else {
          attrs[k] = v
        }
      }
      await api.instances.update(name, activeTab, editTarget.id, {
        version: editTarget.version,
        name: editName !== editTarget.name ? editName : undefined,
        description: editDesc !== editTarget.description ? editDesc : undefined,
        attributes: Object.keys(attrs).length > 0 ? attrs : undefined,
      })
      setEditTarget(null)
      await loadInstances()
    } catch (e) {
      setEditError(e instanceof Error ? e.message : 'Failed to update')
    }
  }

  const handleDelete = async () => {
    if (!name || !activeTab || !deleteTarget) return
    setDeleteError(null)
    try {
      await api.instances.delete(name, activeTab, deleteTarget.id)
      setDeleteTarget(null)
      await loadInstances()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete')
    }
  }

  const getAttrValue = (inst: EntityInstance, attrName: string): string => {
    const av = inst.attributes?.find(a => a.name === attrName)
    return av?.value != null ? String(av.value) : ''
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !catalog) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!catalog) return <PageSection><Alert variant="warning" title="Catalog not found" /></PageSection>

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/catalogs')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Catalogs
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">
        {catalog.name}{' '}
        <Label color={catalog.validation_status === 'valid' ? 'green' : catalog.validation_status === 'invalid' ? 'red' : 'blue'}>
          {catalog.validation_status}
        </Label>
      </Title>
      <p style={{ color: '#6a6e73', marginBottom: '1rem' }}>
        Catalog Version: {catalog.catalog_version_label || catalog.catalog_version_id}
        {catalog.description && ` — ${catalog.description}`}
      </p>

      {pins.length === 0 ? (
        <EmptyState><EmptyStateBody>No entity types pinned in this catalog's version.</EmptyStateBody></EmptyState>
      ) : (
        <Tabs activeKey={activeTab} onSelect={(_e, key) => setActiveTab(String(key))} style={{ marginTop: '1rem' }}>
          {pins.map(pin => (
            <Tab key={pin.entity_type_name} eventKey={pin.entity_type_name} title={<TabTitleText>{pin.entity_type_name}</TabTitleText>}>
              <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
                <Toolbar>
                  <ToolbarContent>
                    {canWrite && (
                      <ToolbarItem>
                        <Button variant="primary" onClick={() => { setCreateOpen(true); setNewInstAttrs({}) }}>
                          Create {pin.entity_type_name}
                        </Button>
                      </ToolbarItem>
                    )}
                    <ToolbarItem>
                      <Button variant="plain" onClick={loadInstances}>Refresh</Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>

                {instLoading ? (
                  <Spinner aria-label="Loading instances" />
                ) : instances.length === 0 ? (
                  <EmptyState><EmptyStateBody>No instances yet. Create one to get started.</EmptyStateBody></EmptyState>
                ) : (
                  <Table aria-label={`${pin.entity_type_name} instances`}>
                    <Thead>
                      <Tr>
                        <Th>Name</Th>
                        <Th>Description</Th>
                        {schemaAttrs.map(attr => (
                          <Th key={attr.name}>{attr.name}</Th>
                        ))}
                        <Th>Version</Th>
                        {canWrite && <Th>Actions</Th>}
                      </Tr>
                    </Thead>
                    <Tbody>
                      {instances.map(inst => (
                        <Tr key={inst.id}>
                          <Td>{inst.name}</Td>
                          <Td>{inst.description}</Td>
                          {schemaAttrs.map(attr => (
                            <Td key={attr.name}>{getAttrValue(inst, attr.name)}</Td>
                          ))}
                          <Td>{inst.version}</Td>
                          {canWrite && (
                            <Td>
                              <Button variant="secondary" size="sm" onClick={() => openEdit(inst)} style={{ marginRight: '0.5rem' }}>Edit</Button>
                              <Button variant="danger" size="sm" onClick={() => setDeleteTarget(inst)}>Delete</Button>
                            </Td>
                          )}
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                )}
                <p style={{ marginTop: '0.5rem' }}>Total: {instTotal}</p>
              </PageSection>
            </Tab>
          ))}
        </Tabs>
      )}

      {/* Create Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={createOpen} onClose={() => { setCreateOpen(false); setCreateError(null) }}>
        <ModalHeader title={`Create ${activeTab}`} />
        <ModalBody>
          {createError && <Alert variant="danger" title={createError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="inst-name">
              <TextInput id="inst-name" value={newInstName} onChange={(_e, v) => setNewInstName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="inst-desc">
              <TextInput id="inst-desc" value={newInstDesc} onChange={(_e, v) => setNewInstDesc(v)} />
            </FormGroup>
            {schemaAttrs.map(attr => (
              <FormGroup key={attr.name} label={`${attr.name}${attr.required ? ' *' : ''}`} fieldId={`attr-${attr.name}`}>
                {attr.type === 'enum' && attr.enum_id && enumValues[attr.enum_id] ? (
                  <EnumSelect
                    id={`attr-${attr.name}`}
                    value={newInstAttrs[attr.name] || ''}
                    options={enumValues[attr.enum_id]}
                    onChange={(v) => setNewInstAttrs(prev => ({ ...prev, [attr.name]: v }))}
                  />
                ) : (
                  <TextInput
                    id={`attr-${attr.name}`}
                    type={attr.type === 'number' ? 'number' : 'text'}
                    value={newInstAttrs[attr.name] || ''}
                    onChange={(_e, v) => setNewInstAttrs(prev => ({ ...prev, [attr.name]: v }))}
                  />
                )}
              </FormGroup>
            ))}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreate} isDisabled={!newInstName.trim()}>Create</Button>
          <Button variant="link" onClick={() => { setCreateOpen(false); setCreateError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Edit Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={editTarget !== null} onClose={() => { setEditTarget(null); setEditError(null) }}>
        <ModalHeader title={`Edit ${editTarget?.name}`} />
        <ModalBody>
          {editError && <Alert variant="danger" title={editError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="edit-name">
              <TextInput id="edit-name" value={editName} onChange={(_e, v) => setEditName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="edit-desc">
              <TextInput id="edit-desc" value={editDesc} onChange={(_e, v) => setEditDesc(v)} />
            </FormGroup>
            {schemaAttrs.map(attr => (
              <FormGroup key={attr.name} label={`${attr.name}${attr.required ? ' *' : ''}`} fieldId={`edit-attr-${attr.name}`}>
                {attr.type === 'enum' && attr.enum_id && enumValues[attr.enum_id] ? (
                  <EnumSelect
                    id={`edit-attr-${attr.name}`}
                    value={editAttrs[attr.name] || ''}
                    options={enumValues[attr.enum_id]}
                    onChange={(v) => setEditAttrs(prev => ({ ...prev, [attr.name]: v }))}
                  />
                ) : (
                  <TextInput
                    id={`edit-attr-${attr.name}`}
                    type={attr.type === 'number' ? 'number' : 'text'}
                    value={editAttrs[attr.name] || ''}
                    onChange={(_e, v) => setEditAttrs(prev => ({ ...prev, [attr.name]: v }))}
                  />
                )}
              </FormGroup>
            ))}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleEdit}>Save</Button>
          <Button variant="link" onClick={() => { setEditTarget(null); setEditError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Instance Modal */}
      <Modal variant={ModalVariant.small} isOpen={deleteTarget !== null} onClose={() => { setDeleteTarget(null); setDeleteError(null) }}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete <strong>{deleteTarget?.name}</strong>? Contained instances will also be deleted.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => { setDeleteTarget(null); setDeleteError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}

// Simple enum select component
function EnumSelect({ id, value, options, onChange }: { id: string; value: string; options: string[]; onChange: (v: string) => void }) {
  const [isOpen, setIsOpen] = useState(false)
  return (
    <Select
      id={id}
      isOpen={isOpen}
      selected={value}
      onSelect={(_e, val) => { onChange(val as string); setIsOpen(false) }}
      onOpenChange={setIsOpen}
      toggle={(ref: React.Ref<MenuToggleElement>) => (
        <MenuToggle ref={ref} onClick={() => setIsOpen(!isOpen)} isExpanded={isOpen} style={{ width: '100%' }}>
          {value || 'Select...'}
        </MenuToggle>
      )}
    >
      {options.map(opt => (
        <SelectOption key={opt} value={opt}>{opt}</SelectOption>
      ))}
    </Select>
  )
}
