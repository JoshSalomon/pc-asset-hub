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
import type { Catalog, CatalogVersionPin, EntityInstance, SnapshotAttribute, SnapshotAssociation, ReferenceDetail, Role } from '../../types'

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

  // Instance detail panel (children + references)
  const [selectedInstance, setSelectedInstance] = useState<EntityInstance | null>(null)
  const [parentName, setParentName] = useState<string>('')
  const [children, setChildren] = useState<EntityInstance[]>([])
  const [childrenLoading, setChildrenLoading] = useState(false)
  const [forwardRefs, setForwardRefs] = useState<ReferenceDetail[]>([])
  const [reverseRefs, setReverseRefs] = useState<ReferenceDetail[]>([])
  const [refsLoading, setRefsLoading] = useState(false)
  const [schemaAssocs, setSchemaAssocs] = useState<SnapshotAssociation[]>([])

  // Add contained instance modal
  const [addChildOpen, setAddChildOpen] = useState(false)
  const [childTypeName, setChildTypeName] = useState('')
  const [addChildMode, setAddChildMode] = useState<'create' | 'adopt'>('create')
  const [newChildName, setNewChildName] = useState('')
  const [newChildDesc, setNewChildDesc] = useState('')
  const [adoptInstanceId, setAdoptInstanceId] = useState('')
  const [availableInstances, setAvailableInstances] = useState<EntityInstance[]>([])
  const [addChildError, setAddChildError] = useState<string | null>(null)
  const [childTypeSelectOpen, setChildTypeSelectOpen] = useState(false)
  const [adoptSelectOpen, setAdoptSelectOpen] = useState(false)
  const [modeSelectOpen, setModeSelectOpen] = useState(false)

  // Link modal
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkTargetId, setLinkTargetId] = useState('')
  const [linkAssocName, setLinkAssocName] = useState('')
  const [linkError, setLinkError] = useState<string | null>(null)
  const [linkAssocSelectOpen, setLinkAssocSelectOpen] = useState(false)
  const [linkTargetSelectOpen, setLinkTargetSelectOpen] = useState(false)
  const [linkTargetInstances, setLinkTargetInstances] = useState<EntityInstance[]>([])

  // Set parent modal (from child side)
  const [setParentOpen, setSetParentOpen] = useState(false)
  const [parentTypeName, setParentTypeName] = useState('')
  const [parentInstanceId, setParentInstanceId] = useState('')
  const [parentInstances, setParentInstances] = useState<EntityInstance[]>([])
  const [setParentError, setSetParentError] = useState<string | null>(null)
  const [parentInstSelectOpen, setParentInstSelectOpen] = useState(false)

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
      setSchemaAssocs(snapshot.associations || [])
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

  const selectInstance = useCallback(async (inst: EntityInstance | null) => {
    setSelectedInstance(inst)
    setParentName('')
    if (!inst || !name || !activeTab) {
      setChildren([])
      setForwardRefs([])
      setReverseRefs([])
      return
    }
    // Resolve parent name if instance is contained
    if (inst.parent_instance_id) {
      try {
        const parent = await api.instances.get(name, activeTab, inst.parent_instance_id)
        setParentName(parent.name)
      } catch { setParentName(inst.parent_instance_id) }
    }
    // Load contained children — find containment associations pointing from this entity type
    setChildrenLoading(true)
    try {
      const containmentAssocs = schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing')
      const allChildren: EntityInstance[] = []
      for (const assoc of containmentAssocs) {
        const childTypeName = assoc.target_entity_type_name
        try {
          const res = await api.instances.listContained(name, activeTab, inst.id, childTypeName)
          allChildren.push(...(res.items || []))
        } catch { /* ignore if child type not found */ }
      }
      setChildren(allChildren)
    } catch {
      setChildren([])
    } finally {
      setChildrenLoading(false)
    }
    // Load references
    setRefsLoading(true)
    try {
      const [fwd, rev] = await Promise.all([
        api.links.forwardRefs(name, activeTab, inst.id),
        api.links.reverseRefs(name, activeTab, inst.id),
      ])
      setForwardRefs(fwd || [])
      setReverseRefs(rev || [])
    } catch {
      setForwardRefs([])
      setReverseRefs([])
    } finally {
      setRefsLoading(false)
    }
  }, [name, activeTab, schemaAssocs])

  // Load available instances when child type selected (for adopt mode)
  const loadAvailableInstances = async (typeName: string) => {
    if (!name || !typeName) { setAvailableInstances([]); return }
    try {
      const res = await api.instances.list(name, typeName)
      // Filter to uncontained instances only
      setAvailableInstances((res.items || []).filter(i => !i.parent_instance_id))
    } catch { setAvailableInstances([]) }
  }

  // Load target instances when association selected in link modal
  const loadLinkTargetInstances = async (assocName: string) => {
    if (!name) return
    const assoc = schemaAssocs.find(a => a.name === assocName && a.direction === 'outgoing')
    if (!assoc) return
    // Find the target entity type name from pins
    const targetPin = pins.find(p => p.entity_type_id === assoc.target_entity_type_id)
    if (!targetPin) return
    try {
      const res = await api.instances.list(name, targetPin.entity_type_name)
      setLinkTargetInstances(res.items || [])
    } catch { setLinkTargetInstances([]) }
  }

  // Load parent instances when parent type selected
  const loadParentInstances = async (typeName: string) => {
    if (!name || !typeName) { setParentInstances([]); return }
    try {
      const res = await api.instances.list(name, typeName)
      setParentInstances(res.items || [])
    } catch { setParentInstances([]) }
  }

  const handleAddChild = async () => {
    if (!name || !activeTab || !selectedInstance || !childTypeName) return
    setAddChildError(null)
    try {
      if (addChildMode === 'adopt' && adoptInstanceId) {
        await api.instances.setParent(name, childTypeName, adoptInstanceId, {
          parent_type: activeTab,
          parent_instance_id: selectedInstance.id,
        })
      } else if (addChildMode === 'create' && newChildName.trim()) {
        await api.instances.createContained(name, activeTab, selectedInstance.id, childTypeName, {
          name: newChildName.trim(),
          description: newChildDesc.trim() || undefined,
        })
      } else {
        return
      }
      setAddChildOpen(false)
      setNewChildName('')
      setNewChildDesc('')
      setChildTypeName('')
      setAdoptInstanceId('')
      setAddChildMode('create')
      await selectInstance(selectedInstance)
    } catch (e) {
      setAddChildError(e instanceof Error ? e.message : 'Failed')
    }
  }

  const handleCreateLink = async () => {
    if (!name || !activeTab || !selectedInstance || !linkTargetId || !linkAssocName) return
    setLinkError(null)
    try {
      await api.links.create(name, activeTab, selectedInstance.id, {
        target_instance_id: linkTargetId,
        association_name: linkAssocName,
      })
      setLinkOpen(false)
      setLinkTargetId('')
      setLinkAssocName('')
      await selectInstance(selectedInstance)
    } catch (e) {
      setLinkError(e instanceof Error ? e.message : 'Failed to create link')
    }
  }

  const handleSetParent = async () => {
    if (!name || !activeTab || !selectedInstance || !parentTypeName) return
    setSetParentError(null)
    try {
      await api.instances.setParent(name, activeTab, selectedInstance.id, {
        parent_type: parentTypeName,
        parent_instance_id: parentInstanceId,
      })
      setSetParentOpen(false)
      setParentTypeName('')
      setParentInstanceId('')
      await loadInstances()
      await selectInstance(selectedInstance)
    } catch (e) {
      setSetParentError(e instanceof Error ? e.message : 'Failed to set parent')
    }
  }

  const handleUnlink = async (linkId: string) => {
    if (!name || !activeTab || !selectedInstance) return
    try {
      await api.links.delete(name, activeTab, selectedInstance.id, linkId)
      await selectInstance(selectedInstance)
    } catch { /* ignore */ }
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
      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>
        Catalog Version: {catalog.catalog_version_label || catalog.catalog_version_id}
        {catalog.description && ` — ${catalog.description}`}
      </p>
      <Button variant="link" isInline component="a" href={`/operational/catalogs/${catalog.name}`} style={{ marginBottom: '1rem' }}>
        Open in Data Viewer →
      </Button>

      {pins.length === 0 ? (
        <EmptyState><EmptyStateBody>No entity types pinned in this catalog's version.</EmptyStateBody></EmptyState>
      ) : (
        <Tabs activeKey={activeTab} onSelect={(_e, key) => { setActiveTab(String(key)); setSelectedInstance(null); setChildren([]); setForwardRefs([]); setReverseRefs([]) }} style={{ marginTop: '1rem' }}>
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
                        <Th>Actions</Th>
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
                          <Td>
                            <Button variant="link" size="sm" onClick={() => selectInstance(selectedInstance?.id === inst.id ? null : inst)}>
                              {selectedInstance?.id === inst.id ? 'Hide Details' : 'Details'}
                            </Button>
                            {canWrite && (
                              <>
                                <Button variant="secondary" size="sm" onClick={() => openEdit(inst)} style={{ marginLeft: '0.5rem' }}>Edit</Button>
                                <Button variant="danger" size="sm" onClick={() => setDeleteTarget(inst)} style={{ marginLeft: '0.5rem' }}>Delete</Button>
                              </>
                            )}
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                )}
                {/* Instance Detail Panel */}
                {selectedInstance && (
                  <div style={{ border: '1px solid #d2d2d2', padding: '1rem', marginTop: '1rem', borderRadius: '4px' }}>
                    <Title headingLevel="h4">Details: {selectedInstance.name}</Title>
                    {selectedInstance.parent_instance_id && (
                      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>Contained by: {parentName || selectedInstance.parent_instance_id}</p>
                    )}
                    {canWrite && schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'incoming').length > 0 && (
                      <Button variant="secondary" size="sm" onClick={() => {
                        setSetParentError(null)
                        setParentInstanceId('')
                        // Auto-select the container type (there should be exactly one)
                        const incomingContainment = schemaAssocs.find(a => a.type === 'containment' && a.direction === 'incoming')
                        if (incomingContainment?.source_entity_type_name) {
                          setParentTypeName(incomingContainment.source_entity_type_name)
                          loadParentInstances(incomingContainment.source_entity_type_name)
                        }
                        setSetParentOpen(true)
                      }} style={{ marginBottom: '0.5rem' }}>
                        Set Container
                      </Button>
                    )}

                    {/* Children Section */}
                    <div style={{ marginTop: '1rem' }}>
                      <Title headingLevel="h5">Contained Instances</Title>
                      {canWrite && schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing').length > 0 && (
                        <Button variant="secondary" size="sm" onClick={() => {
                          setAddChildError(null)
                          setAddChildMode('create')
                          setNewChildName('')
                          setNewChildDesc('')
                          setAdoptInstanceId('')
                          const containmentAssocs = schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing')
                          if (containmentAssocs.length === 1) {
                            setChildTypeName(containmentAssocs[0].target_entity_type_name)
                            loadAvailableInstances(containmentAssocs[0].target_entity_type_name)
                          } else {
                            setChildTypeName('')
                            setAvailableInstances([])
                          }
                          setAddChildOpen(true)
                        }} style={{ marginBottom: '0.5rem' }}>
                          Add Contained Instance
                        </Button>
                      )}
                      {childrenLoading ? (
                        <Spinner aria-label="Loading children" size="md" />
                      ) : children.length === 0 ? (
                        <p style={{ color: '#6a6e73' }}>No contained instances.</p>
                      ) : (
                        <Table aria-label="Contained instances" variant="compact">
                          <Thead><Tr><Th>Name</Th><Th>Entity Type</Th><Th>Description</Th></Tr></Thead>
                          <Tbody>
                            {children.map(child => (
                              <Tr key={child.id}>
                                <Td>{child.name}</Td>
                                <Td>{schemaAssocs.find(a => a.type === 'containment' && a.direction === 'outgoing' && a.target_entity_type_id === child.entity_type_id)?.target_entity_type_name || child.entity_type_id}</Td>
                                <Td>{child.description}</Td>
                              </Tr>
                            ))}
                          </Tbody>
                        </Table>
                      )}
                    </div>

                    {/* References Section */}
                    <div style={{ marginTop: '1rem' }}>
                      <Title headingLevel="h5">References</Title>
                      {canWrite && schemaAssocs.filter(a => a.type !== 'containment' && a.direction === 'outgoing').length > 0 && (
                        <Button variant="secondary" size="sm" onClick={() => { setLinkOpen(true); setLinkError(null) }} style={{ marginBottom: '0.5rem' }}>
                          Link to Instance
                        </Button>
                      )}
                      {refsLoading ? (
                        <Spinner aria-label="Loading references" size="md" />
                      ) : (
                        <>
                          {forwardRefs.length > 0 && (
                            <>
                              <p><strong>Forward References</strong></p>
                              <Table aria-label="Forward references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Target</Th><Th>Entity Type</Th>{canWrite && <Th screenReaderText="Actions" />}</Tr></Thead>
                                <Tbody>
                                  {forwardRefs.map(ref => (
                                    <Tr key={ref.link_id}>
                                      <Td>{ref.association_name}</Td>
                                      <Td>{ref.association_type}</Td>
                                      <Td>{ref.instance_name}</Td>
                                      <Td>{ref.entity_type_name}</Td>
                                      {canWrite && (
                                        <Td><Button variant="link" size="sm" onClick={() => handleUnlink(ref.link_id)}>Unlink</Button></Td>
                                      )}
                                    </Tr>
                                  ))}
                                </Tbody>
                              </Table>
                            </>
                          )}
                          {reverseRefs.length > 0 && (
                            <>
                              <p style={{ marginTop: '0.5rem' }}><strong>Referenced By</strong></p>
                              <Table aria-label="Reverse references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Source</Th><Th>Entity Type</Th></Tr></Thead>
                                <Tbody>
                                  {reverseRefs.map(ref => (
                                    <Tr key={ref.link_id}>
                                      <Td>{ref.association_name}</Td>
                                      <Td>{ref.association_type}</Td>
                                      <Td>{ref.instance_name}</Td>
                                      <Td>{ref.entity_type_name}</Td>
                                    </Tr>
                                  ))}
                                </Tbody>
                              </Table>
                            </>
                          )}
                          {forwardRefs.length === 0 && reverseRefs.length === 0 && (
                            <p style={{ color: '#6a6e73' }}>No references.</p>
                          )}
                        </>
                      )}
                    </div>
                  </div>
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

      {/* Add Contained Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={addChildOpen} onClose={() => { setAddChildOpen(false); setAddChildError(null) }}>
        <ModalHeader title="Add Contained Instance" />
        <ModalBody>
          {addChildError && <Alert variant="danger" title={addChildError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Child Entity Type" isRequired fieldId="child-type">
              <Select
                id="child-type"
                isOpen={childTypeSelectOpen}
                selected={childTypeName}
                onSelect={(_e, val) => {
                  const v = val as string
                  setChildTypeName(v)
                  setChildTypeSelectOpen(false)
                  loadAvailableInstances(v)
                }}
                onOpenChange={setChildTypeSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setChildTypeSelectOpen(!childTypeSelectOpen)} isExpanded={childTypeSelectOpen} style={{ width: '100%' }}>
                    {childTypeName || 'Select child type...'}
                  </MenuToggle>
                )}
              >
                {schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing').map(a => (
                  <SelectOption key={a.target_entity_type_name} value={a.target_entity_type_name}>
                    {a.target_entity_type_name}
                  </SelectOption>
                ))}
              </Select>
            </FormGroup>
            <FormGroup label="Mode" fieldId="child-mode">
              {availableInstances.length > 0 ? (
                <Select
                  id="child-mode"
                  isOpen={modeSelectOpen}
                  selected={addChildMode}
                  onSelect={(_e, val) => { setAddChildMode(val as 'create' | 'adopt'); setModeSelectOpen(false) }}
                  onOpenChange={setModeSelectOpen}
                  toggle={(ref: React.Ref<MenuToggleElement>) => (
                    <MenuToggle ref={ref} onClick={() => setModeSelectOpen(!modeSelectOpen)} isExpanded={modeSelectOpen} style={{ width: '100%' }}>
                      {addChildMode === 'create' ? 'Create New' : 'Adopt Existing'}
                    </MenuToggle>
                  )}
                >
                  <SelectOption value="create">Create New</SelectOption>
                  <SelectOption value="adopt">Adopt Existing</SelectOption>
                </Select>
              ) : (
                <MenuToggle isDisabled style={{ width: '100%' }}>Create New</MenuToggle>
              )}
            </FormGroup>
            {(addChildMode === 'create' || availableInstances.length === 0) ? (
              <>
                <FormGroup label="Name" isRequired fieldId="child-name">
                  <TextInput id="child-name" value={newChildName} onChange={(_e, v) => setNewChildName(v)} isRequired />
                </FormGroup>
                <FormGroup label="Description" fieldId="child-desc">
                  <TextInput id="child-desc" value={newChildDesc} onChange={(_e, v) => setNewChildDesc(v)} />
                </FormGroup>
              </>
            ) : (
              <FormGroup label="Select Instance" isRequired fieldId="adopt-instance">
                <Select
                  id="adopt-instance"
                  isOpen={adoptSelectOpen}
                  selected={adoptInstanceId}
                  onSelect={(_e, val) => { setAdoptInstanceId(val as string); setAdoptSelectOpen(false) }}
                  onOpenChange={setAdoptSelectOpen}
                  toggle={(ref: React.Ref<MenuToggleElement>) => (
                    <MenuToggle ref={ref} onClick={() => setAdoptSelectOpen(!adoptSelectOpen)} isExpanded={adoptSelectOpen} style={{ width: '100%' }}>
                      {availableInstances.find(i => i.id === adoptInstanceId)?.name || 'Select instance...'}
                    </MenuToggle>
                  )}
                >
                  {availableInstances.map(inst => (
                    <SelectOption key={inst.id} value={inst.id}>{inst.name}</SelectOption>
                  ))}
                </Select>
              </FormGroup>
            )}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleAddChild}
            isDisabled={!childTypeName || (addChildMode === 'create' ? !newChildName.trim() : !adoptInstanceId)}>
            {addChildMode === 'create' ? 'Create' : 'Adopt'}
          </Button>
          <Button variant="link" onClick={() => { setAddChildOpen(false); setAddChildError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Link to Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={linkOpen} onClose={() => { setLinkOpen(false); setLinkError(null) }}>
        <ModalHeader title="Link to Instance" />
        <ModalBody>
          {linkError && <Alert variant="danger" title={linkError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Association" isRequired fieldId="link-assoc">
              <Select
                id="link-assoc"
                isOpen={linkAssocSelectOpen}
                selected={linkAssocName}
                onSelect={(_e, val) => {
                  const v = val as string
                  setLinkAssocName(v)
                  setLinkAssocSelectOpen(false)
                  setLinkTargetId('')
                  loadLinkTargetInstances(v)
                }}
                onOpenChange={setLinkAssocSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setLinkAssocSelectOpen(!linkAssocSelectOpen)} isExpanded={linkAssocSelectOpen} style={{ width: '100%' }}>
                    {linkAssocName || 'Select association...'}
                  </MenuToggle>
                )}
              >
                {schemaAssocs.filter(a => a.type !== 'containment' && a.direction === 'outgoing').map(a => (
                  <SelectOption key={a.name} value={a.name}>
                    {a.name} → {a.target_entity_type_name}
                  </SelectOption>
                ))}
              </Select>
            </FormGroup>
            <FormGroup label="Target Instance" isRequired fieldId="link-target">
              <Select
                id="link-target"
                isOpen={linkTargetSelectOpen}
                selected={linkTargetId}
                onSelect={(_e, val) => { setLinkTargetId(val as string); setLinkTargetSelectOpen(false) }}
                onOpenChange={setLinkTargetSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setLinkTargetSelectOpen(!linkTargetSelectOpen)} isExpanded={linkTargetSelectOpen} style={{ width: '100%' }}>
                    {linkTargetInstances.find(i => i.id === linkTargetId)?.name || 'Select target instance...'}
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
          <Button variant="primary" onClick={handleCreateLink} isDisabled={!linkAssocName || !linkTargetId}>Link</Button>
          <Button variant="link" onClick={() => { setLinkOpen(false); setLinkError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Set Container Modal (from child side) */}
      <Modal variant={ModalVariant.medium} isOpen={setParentOpen} onClose={() => { setSetParentOpen(false); setSetParentError(null) }}>
        <ModalHeader title={`Set Container for ${selectedInstance?.name}`} />
        <ModalBody>
          {setParentError && <Alert variant="danger" title={setParentError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Container Type" fieldId="parent-type">
              <TextInput id="parent-type" value={parentTypeName} isDisabled aria-label="Container type" />
            </FormGroup>
            <FormGroup label="Container Instance" isRequired fieldId="parent-instance">
              <Select
                id="parent-instance"
                isOpen={parentInstSelectOpen}
                selected={parentInstanceId}
                onSelect={(_e, val) => { setParentInstanceId(val as string); setParentInstSelectOpen(false) }}
                onOpenChange={setParentInstSelectOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setParentInstSelectOpen(!parentInstSelectOpen)} isExpanded={parentInstSelectOpen} style={{ width: '100%' }}>
                    {parentInstances.find(i => i.id === parentInstanceId)?.name || 'Select container...'}
                  </MenuToggle>
                )}
              >
                {parentInstances.map(inst => (
                  <SelectOption key={inst.id} value={inst.id}>{inst.name}</SelectOption>
                ))}
              </Select>
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleSetParent} isDisabled={!parentTypeName || !parentInstanceId}>Set Container</Button>
          <Button variant="danger" onClick={() => {
            if (selectedInstance && name && activeTab) {
              api.instances.setParent(name, activeTab, selectedInstance.id, { parent_type: '', parent_instance_id: '' })
                .then(() => { setSetParentOpen(false); loadInstances(); selectInstance(selectedInstance) })
                .catch(() => {})
            }
          }} isDisabled={!selectedInstance?.parent_instance_id}>Remove Container</Button>
          <Button variant="link" onClick={() => { setSetParentOpen(false); setSetParentError(null) }}>Cancel</Button>
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
