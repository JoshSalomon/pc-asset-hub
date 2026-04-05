import { useState, useEffect } from 'react'
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
  Alert,
  Label,
  EmptyState,
  EmptyStateBody,
  Spinner,
  TextInput,
  MenuToggle,
  Select,
  SelectOption,
  SelectList,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, CatalogVersion, EntityInstance, SnapshotAttribute, Role } from '../../types'
import { useValidation } from '../../hooks/useValidation'
import ValidationResults from '../../components/ValidationResults'
import { useCatalogData } from '../../hooks/useCatalogData'
import { useInstances } from '../../hooks/useInstances'
import { useInstanceDetail } from '../../hooks/useInstanceDetail'
import CreateInstanceModal from '../../components/CreateInstanceModal'
import EditInstanceModal from '../../components/EditInstanceModal'
import AddChildModal from '../../components/AddChildModal'
import type { AddChildCreateData, AddChildAdoptData } from '../../components/AddChildModal'
import LinkModal from '../../components/LinkModal'
import SetParentModal from '../../components/SetParentModal'
import CopyCatalogModal from '../../components/CopyCatalogModal'
import ReplaceCatalogModal from '../../components/ReplaceCatalogModal'
import { buildTypedAttrs } from '../../utils/buildTypedAttrs'
import { useCatalogDiagram } from '../../hooks/useCatalogDiagram'
import EntityTypeDiagram from '../../components/EntityTypeDiagram'

export default function CatalogDetailPage({ role }: { role: Role }) {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()

  const {
    catalog, loading, error, setError,
    pins, activeTab, setActiveTab,
    schemaAttrs, schemaAssocs, enumValues,
    loadCatalog,
  } = useCatalogData(name, role)

  const inst = useInstances(name, activeTab, schemaAttrs, role)
  const detail = useInstanceDetail(name, activeTab, schemaAssocs)

  useEffect(() => { inst.loadInstances() }, [inst.loadInstances])

  const validation = useValidation(name, loadCatalog)
  const diagram = useCatalogDiagram(catalog?.catalog_version_id)

  useEffect(() => {
    if (activeTab === '__diagram__') diagram.loadDiagram()
  }, [activeTab, diagram.loadDiagram])

  const canWrite = role === 'RW' || role === 'Admin' || role === 'SuperAdmin'
  const isAdmin = role === 'Admin' || role === 'SuperAdmin'
  const canMutate = canWrite && (!catalog?.published || role === 'SuperAdmin')
  const canValidate = canMutate

  // Add contained instance modal state
  const [addChildOpen, setAddChildOpen] = useState(false)
  const [addChildError, setAddChildError] = useState<string | null>(null)
  const [childSchemaAttrs, setChildSchemaAttrs] = useState<SnapshotAttribute[]>([])
  const [childEnumValues, setChildEnumValues] = useState<Record<string, string[]>>({})
  const [availableInstances, setAvailableInstances] = useState<EntityInstance[]>([])
  const [initialChildType, setInitialChildType] = useState('')

  // Link modal state
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkError, setLinkError] = useState<string | null>(null)
  const [linkTargetInstances, setLinkTargetInstances] = useState<EntityInstance[]>([])

  // Set parent modal state
  const [setParentOpen, setSetParentOpen] = useState(false)
  const [parentTypeName, setParentTypeName] = useState('')
  const [parentInstances, setParentInstances] = useState<EntityInstance[]>([])
  const [setParentError, setSetParentError] = useState<string | null>(null)

  // Copy catalog modal
  const [copyOpen, setCopyOpen] = useState(false)
  const [copyError, setCopyError] = useState<string | null>(null)
  const [copyLoading, setCopyLoading] = useState(false)

  // Description editing
  const [editingDesc, setEditingDesc] = useState(false)
  const [editDescValue, setEditDescValue] = useState('')

  // CV selector
  const [cvList, setCvList] = useState<CatalogVersion[]>([])
  const [cvSelectOpen, setCvSelectOpen] = useState(false)

  // Replace catalog modal
  const [replaceOpen, setReplaceOpen] = useState(false)
  const [replaceError, setReplaceError] = useState<string | null>(null)
  const [replaceLoading, setReplaceLoading] = useState(false)
  const [availableCatalogs, setAvailableCatalogs] = useState<Catalog[]>([])

  // Load available instances when child type selected (for adopt mode)
  const loadAvailableInstances = async (typeName: string) => {
    if (!name || !typeName) { setAvailableInstances([]); return }
    try {
      const res = await api.instances.list(name, typeName)
      // Filter to uncontained instances only
      setAvailableInstances((res.items || []).filter((i: EntityInstance) => !i.parent_instance_id))
    } catch { setAvailableInstances([]) }
  }

  // Load child type schema attributes when child type is selected
  const loadChildSchema = async (typeName: string) => {
    if (!typeName || !pins.length) { setChildSchemaAttrs([]); return }
    const pin = pins.find(p => p.entity_type_name === typeName)
    if (!pin) { setChildSchemaAttrs([]); return }
    try {
      const snapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
      setChildSchemaAttrs(snapshot.attributes || [])
      // Load enum values for enum attributes
      const cache: Record<string, string[]> = {}
      for (const attr of snapshot.attributes || []) {
        if (attr.type === 'enum' && attr.enum_id && !cache[attr.enum_id]) {
          try {
            const res = await api.enums.listValues(attr.enum_id)
            cache[attr.enum_id] = (res.items || []).map((v: { value: string }) => v.value)
          } catch { /* ignore */ }
        }
      }
      setChildEnumValues(cache)
    } catch { setChildSchemaAttrs([]) }
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

  const handleAddChild = async (childType: string, mode: 'create' | 'adopt', data: AddChildCreateData | AddChildAdoptData) => {
    if (!name || !activeTab || !detail.selectedInstance || !childType) return
    setAddChildError(null)
    try {
      if (mode === 'adopt') {
        const adoptData = data as AddChildAdoptData
        await api.instances.setParent(name, childType, adoptData.adoptInstanceId, {
          parent_type: activeTab,
          parent_instance_id: detail.selectedInstance.id,
        })
      } else if (mode === 'create') {
        const createData = data as AddChildCreateData
        if (!createData.name.trim()) return
        const childAttrs = buildTypedAttrs(createData.attrs, childSchemaAttrs)
        await api.instances.createContained(name, activeTab, detail.selectedInstance.id, childType, {
          name: createData.name.trim(),
          description: createData.description.trim() || undefined,
          ...(Object.keys(childAttrs).length > 0 ? { attributes: childAttrs } : {}),
        })
      } else {
        return
      }
      setAddChildOpen(false)
      await detail.selectInstance(detail.selectedInstance)
    } catch (e) {
      setAddChildError(e instanceof Error ? e.message : 'Failed')
    }
  }

  const handleCreateLink = async (targetId: string, assocName: string) => {
    if (!name || !activeTab || !detail.selectedInstance || !targetId || !assocName) return
    setLinkError(null)
    try {
      await api.links.create(name, activeTab, detail.selectedInstance.id, {
        target_instance_id: targetId,
        association_name: assocName,
      })
      setLinkOpen(false)
      await detail.selectInstance(detail.selectedInstance)
    } catch (e) {
      setLinkError(e instanceof Error ? e.message : 'Failed to create link')
    }
  }

  const handleSetParent = async (pType: string, pId: string) => {
    if (!name || !activeTab || !detail.selectedInstance || !pType) return
    setSetParentError(null)
    try {
      await api.instances.setParent(name, activeTab, detail.selectedInstance.id, {
        parent_type: pType,
        parent_instance_id: pId,
      })
      setSetParentOpen(false)
      await inst.loadInstances()
      await detail.selectInstance(detail.selectedInstance)
    } catch (e) {
      setSetParentError(e instanceof Error ? e.message : 'Failed to set parent')
    }
  }

  const handleUnlink = async (linkId: string) => {
    if (!name || !activeTab || !detail.selectedInstance) return
    try {
      await api.links.delete(name, activeTab, detail.selectedInstance.id, linkId)
      await detail.selectInstance(detail.selectedInstance)
    } catch { /* ignore */ }
  }

  const getAttrValue = (instance: EntityInstance, attrName: string): string => {
    const av = instance.attributes?.find(a => a.name === attrName)
    return av?.value != null ? String(av.value) : ''
  }

  const handleCopy = async (copyName: string, copyDesc: string) => {
    if (!catalog) return
    setCopyError(null)
    setCopyLoading(true)
    try {
      await api.catalogs.copy({ source: catalog.name, name: copyName, description: copyDesc || undefined })
      setCopyOpen(false)
      navigate(`/schema/catalogs/${copyName}`)
    } catch (e) {
      setCopyError(e instanceof Error ? e.message : 'Failed to copy catalog')
    } finally {
      setCopyLoading(false)
    }
  }

  const handleReplace = async (target: string, archiveNameVal: string) => {
    if (!catalog) return
    setReplaceError(null)
    setReplaceLoading(true)
    try {
      await api.catalogs.replace({ source: catalog.name, target, archive_name: archiveNameVal || undefined })
      setReplaceOpen(false)
      navigate('/schema/catalogs')
    } catch (e) {
      setReplaceError(e instanceof Error ? e.message : 'Failed to replace catalog')
    } finally {
      setReplaceLoading(false)
    }
  }

  const handleSaveDescription = async () => {
    if (!name) return
    setAuthRole(role)
    try {
      await api.catalogs.update(name, { description: editDescValue })
      setEditingDesc(false)
      await loadCatalog()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update description')
    }
  }

  const handleChangeCv = async (cvId: string) => {
    if (!name || !cvId) return
    setCvSelectOpen(false)
    setAuthRole(role)
    try {
      await api.catalogs.update(name, { catalog_version_id: cvId })
      await loadCatalog()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update catalog version')
    }
  }

  const [cvListLoading, setCvListLoading] = useState(false)
  const handleOpenCvSelector = async () => {
    if (cvListLoading) return
    setCvListLoading(true)
    setAuthRole(role)
    try {
      const res = await api.catalogVersions.list()
      setCvList(res.items || [])
    } catch { /* ignore */ } finally {
      setCvListLoading(false)
    }
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !catalog) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!catalog) return <PageSection><Alert variant="warning" title="Catalog not found" /></PageSection>

  const canPublishOrReplace = isAdmin && !catalog.published && catalog.validation_status === 'valid'

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/schema/catalogs')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Catalogs
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">
        {catalog.name}{' '}
        <Label color={catalog.validation_status === 'valid' ? 'green' : catalog.validation_status === 'invalid' ? 'red' : 'blue'}>
          {catalog.validation_status}
        </Label>
        {catalog.published && (
          <Label color="purple" style={{ marginLeft: '0.5rem' }}>published</Label>
        )}
      </Title>
      <DescriptionList isHorizontal style={{ marginBottom: '0.5rem' }}>
        <DescriptionListGroup>
          <DescriptionListTerm>Catalog Version</DescriptionListTerm>
          <DescriptionListDescription>
            {isAdmin && !catalog.published ? (
              <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                <Select
                  isOpen={cvSelectOpen}
                  onOpenChange={(open) => { setCvSelectOpen(open); if (open) handleOpenCvSelector() }}
                  toggle={(toggleRef) => (
                    <MenuToggle ref={toggleRef} onClick={() => { if (!cvSelectOpen) handleOpenCvSelector(); setCvSelectOpen(!cvSelectOpen) }} isExpanded={cvSelectOpen} aria-label="Select catalog version">
                      {catalog.catalog_version_label || catalog.catalog_version_id}
                    </MenuToggle>
                  )}
                  onSelect={(_e, val) => handleChangeCv(String(val))}
                  selected={catalog.catalog_version_id}
                >
                  <SelectList>
                    {cvList.map(cv => (
                      <SelectOption key={cv.id} value={cv.id}>{cv.version_label}</SelectOption>
                    ))}
                  </SelectList>
                </Select>
              </div>
            ) : (
              <span>{catalog.catalog_version_label || catalog.catalog_version_id}</span>
            )}
          </DescriptionListDescription>
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
                  style={{ maxWidth: '300px' }}
                />
                <Button variant="primary" size="sm" onClick={handleSaveDescription}>Save</Button>
                <Button variant="link" size="sm" onClick={() => setEditingDesc(false)}>Cancel</Button>
              </div>
            ) : (
              <>
                {catalog.description || <span style={{ color: '#6a6e73' }}>No description</span>}
                {canMutate && (
                  <Button variant="link" size="sm" onClick={() => { setEditDescValue(catalog.description || ''); setEditingDesc(true) }} style={{ marginLeft: '0.5rem' }} aria-label="Edit description">Edit</Button>
                )}
              </>
            )}
          </DescriptionListDescription>
        </DescriptionListGroup>
      </DescriptionList>

      {catalog.published && !isAdmin && (
        <Alert variant="info" title="This catalog is published. Editing requires SuperAdmin privileges." isInline style={{ marginBottom: '1rem' }} />
      )}

      <div style={{ display: 'flex', gap: '1rem', alignItems: 'center', marginBottom: '1rem' }}>
        <Button variant="link" isInline onClick={() => navigate(`/catalogs/${catalog.name}`)}>
          Open in Data Viewer →
        </Button>
        {canValidate && (
          <Button variant="secondary" onClick={validation.validate} isLoading={validation.validating} isDisabled={validation.validating}>
            Validate
          </Button>
        )}
        {canPublishOrReplace && (
          <Button variant="primary" onClick={async () => {
            try { await api.catalogs.publish(catalog.name); await loadCatalog() }
            catch (e) { setError(e instanceof Error ? e.message : 'Failed to publish') }
          }}>
            Publish
          </Button>
        )}
        {isAdmin && catalog.published && (
          <Button variant="warning" onClick={async () => {
            try { await api.catalogs.unpublish(catalog.name); await loadCatalog() }
            catch (e) { setError(e instanceof Error ? e.message : 'Failed to unpublish') }
          }}>
            Unpublish
          </Button>
        )}
        {canMutate && (
          <Button variant="secondary" onClick={() => { setCopyOpen(true); setCopyError(null) }}>
            Copy
          </Button>
        )}
        {canPublishOrReplace && (
          <Button variant="secondary" onClick={async () => {
            setReplaceError(null)
            try {
              const res = await api.catalogs.list()
              setAvailableCatalogs((res.items || []).filter((c: Catalog) => c.name !== catalog.name))
            } catch { /* ignore */ }
            setReplaceOpen(true)
          }}>
            Replace
          </Button>
        )}
      </div>

      <ValidationResults errors={validation.errors} ran={validation.ran} error={validation.error} />

      {pins.length === 0 ? (
        <EmptyState><EmptyStateBody>No entity types pinned in this catalog's version.</EmptyStateBody></EmptyState>
      ) : (
        <Tabs activeKey={activeTab} onSelect={(_e, key) => { setActiveTab(String(key)); detail.clearSelection() }} style={{ marginTop: '1rem' }}>
          {[...pins.map(pin => (
            <Tab key={pin.entity_type_name} eventKey={pin.entity_type_name} title={<TabTitleText>{pin.entity_type_name}</TabTitleText>}>
              <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
                <Toolbar>
                  <ToolbarContent>
                    {canMutate && (
                      <ToolbarItem>
                        <Button variant="primary" onClick={inst.openCreate}>
                          Create {pin.entity_type_name}
                        </Button>
                      </ToolbarItem>
                    )}
                    <ToolbarItem>
                      <Button variant="plain" onClick={inst.loadInstances}>Refresh</Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>

                {inst.instLoading ? (
                  <Spinner aria-label="Loading instances" />
                ) : inst.instances.length === 0 ? (
                  <EmptyState><EmptyStateBody>No instances yet. Create one to get started.</EmptyStateBody></EmptyState>
                ) : (
                  <Table aria-label={`${pin.entity_type_name} instances`}>
                    <Thead>
                      <Tr>
                        <Th>Name</Th>
                        <Th>Description</Th>
                        {schemaAttrs.filter(attr => !attr.system).map(attr => (
                          <Th key={attr.name}>{attr.name}</Th>
                        ))}
                        <Th>Version</Th>
                        <Th>Actions</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {inst.instances.map(instance => (
                        <Tr key={instance.id}>
                          <Td>{instance.name}</Td>
                          <Td>{instance.description}</Td>
                          {schemaAttrs.filter(attr => !attr.system).map(attr => (
                            <Td key={attr.name}>{getAttrValue(instance, attr.name)}</Td>
                          ))}
                          <Td>{instance.version}</Td>
                          <Td>
                            <Button variant="link" size="sm" onClick={() => detail.selectInstance(detail.selectedInstance?.id === instance.id ? null : instance)}>
                              {detail.selectedInstance?.id === instance.id ? 'Hide Details' : 'Details'}
                            </Button>
                            {canMutate && (
                              <>
                                <Button variant="secondary" size="sm" onClick={() => inst.openEdit(instance)} style={{ marginLeft: '0.5rem' }}>Edit</Button>
                                <Button variant="danger" size="sm" onClick={() => inst.openDelete(instance)} style={{ marginLeft: '0.5rem' }}>Delete</Button>
                              </>
                            )}
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                )}
                {/* Instance Detail Panel */}
                {detail.selectedInstance && (
                  <div style={{ border: '1px solid #d2d2d2', padding: '1rem', marginTop: '1rem', borderRadius: '4px' }}>
                    <Title headingLevel="h4">Details: {detail.selectedInstance.name}</Title>
                    {detail.selectedInstance.parent_instance_id && (
                      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>Contained by: {detail.parentName || detail.selectedInstance.parent_instance_id}</p>
                    )}
                    {canMutate && schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'incoming').length > 0 && (
                      <Button variant="secondary" size="sm" onClick={() => {
                        setSetParentError(null)
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
                      {canMutate && schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing').length > 0 && (
                        <Button variant="secondary" size="sm" onClick={() => {
                          setAddChildError(null)
                          setAvailableInstances([])
                          setChildSchemaAttrs([])
                          const containmentAssocs = schemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing')
                          if (containmentAssocs.length === 1) {
                            setInitialChildType(containmentAssocs[0].target_entity_type_name)
                            loadAvailableInstances(containmentAssocs[0].target_entity_type_name)
                            loadChildSchema(containmentAssocs[0].target_entity_type_name)
                          } else {
                            setInitialChildType('')
                          }
                          setAddChildOpen(true)
                        }} style={{ marginBottom: '0.5rem' }}>
                          Add Contained Instance
                        </Button>
                      )}
                      {detail.childrenLoading ? (
                        <Spinner aria-label="Loading children" size="md" />
                      ) : detail.children.length === 0 ? (
                        <p style={{ color: '#6a6e73' }}>No contained instances.</p>
                      ) : (
                        <Table aria-label="Contained instances" variant="compact">
                          <Thead><Tr><Th>Name</Th><Th>Entity Type</Th><Th>Description</Th></Tr></Thead>
                          <Tbody>
                            {detail.children.map(child => (
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
                      {canMutate && schemaAssocs.filter(a => a.type !== 'containment' && a.direction === 'outgoing').length > 0 && (
                        <Button variant="secondary" size="sm" onClick={() => { setLinkOpen(true); setLinkError(null) }} style={{ marginBottom: '0.5rem' }}>
                          Link to Instance
                        </Button>
                      )}
                      {detail.refsLoading ? (
                        <Spinner aria-label="Loading references" size="md" />
                      ) : (
                        <>
                          {detail.forwardRefs.length > 0 && (
                            <>
                              <p><strong>Forward References</strong></p>
                              <Table aria-label="Forward references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Target</Th><Th>Entity Type</Th>{canMutate && <Th screenReaderText="Actions" />}</Tr></Thead>
                                <Tbody>
                                  {detail.forwardRefs.map(ref => (
                                    <Tr key={ref.link_id}>
                                      <Td>{ref.association_name}</Td>
                                      <Td>{ref.association_type}</Td>
                                      <Td>{ref.instance_name}</Td>
                                      <Td>{ref.entity_type_name}</Td>
                                      {canMutate && (
                                        <Td><Button variant="link" size="sm" onClick={() => handleUnlink(ref.link_id)}>Unlink</Button></Td>
                                      )}
                                    </Tr>
                                  ))}
                                </Tbody>
                              </Table>
                            </>
                          )}
                          {detail.reverseRefs.length > 0 && (
                            <>
                              <p style={{ marginTop: '0.5rem' }}><strong>Referenced By</strong></p>
                              <Table aria-label="Reverse references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Source</Th><Th>Entity Type</Th></Tr></Thead>
                                <Tbody>
                                  {detail.reverseRefs.map(ref => (
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
                          {detail.forwardRefs.length === 0 && detail.reverseRefs.length === 0 && (
                            <p style={{ color: '#6a6e73' }}>No references.</p>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                )}

                <p style={{ marginTop: '0.5rem' }}>Total: {inst.instTotal}</p>
              </PageSection>
            </Tab>
          )),
          <Tab key="__diagram__" eventKey="__diagram__" title={<TabTitleText>Model Diagram</TabTitleText>}>
            <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
              {diagram.diagramError && (
                <Alert variant="danger" title={diagram.diagramError} isInline style={{ marginBottom: '1rem' }} />
              )}
              {diagram.diagramLoading ? (
                <Spinner aria-label="Loading diagram" />
              ) : diagram.diagramData.length === 0 && !diagram.diagramError ? (
                <EmptyState><EmptyStateBody>No model diagram available. The catalog version has no pinned entity types.</EmptyStateBody></EmptyState>
              ) : (
                <EntityTypeDiagram entityTypes={diagram.diagramData} />
              )}
            </PageSection>
          </Tab>,
          ]}
        </Tabs>
      )}

      {/* Create Instance Modal */}
      <CreateInstanceModal
        isOpen={inst.createOpen}
        onClose={inst.closeCreate}
        entityTypeName={activeTab}
        schemaAttrs={schemaAttrs}
        enumValues={enumValues}
        onSubmit={inst.handleCreate}
        error={inst.createError}
      />

      {/* Edit Instance Modal */}
      <EditInstanceModal
        instance={inst.editTarget}
        onClose={inst.closeEdit}
        schemaAttrs={schemaAttrs}
        enumValues={enumValues}
        onSubmit={inst.handleEdit}
        error={inst.editError}
      />

      {/* Add Contained Instance Modal */}
      <AddChildModal
        isOpen={addChildOpen}
        onClose={() => { setAddChildOpen(false); setAddChildError(null) }}
        schemaAssocs={schemaAssocs}
        childSchemaAttrs={childSchemaAttrs}
        childEnumValues={childEnumValues}
        availableInstances={availableInstances}
        onChildTypeChange={(v) => { loadAvailableInstances(v); loadChildSchema(v) }}
        onSubmit={handleAddChild}
        error={addChildError}
        initialChildType={initialChildType}
      />

      {/* Link to Instance Modal */}
      <LinkModal
        isOpen={linkOpen}
        onClose={() => { setLinkOpen(false); setLinkError(null) }}
        schemaAssocs={schemaAssocs}
        linkTargetInstances={linkTargetInstances}
        onAssocChange={loadLinkTargetInstances}
        onSubmit={handleCreateLink}
        error={linkError}
      />

      {/* Set Container Modal (from child side) */}
      <SetParentModal
        isOpen={setParentOpen}
        onClose={() => { setSetParentOpen(false); setSetParentError(null) }}
        instanceName={detail.selectedInstance?.name}
        parentTypeName={parentTypeName}
        parentInstances={parentInstances}
        hasParent={!!detail.selectedInstance?.parent_instance_id}
        onSubmit={handleSetParent}
        onRemoveParent={() => {
          if (detail.selectedInstance && name && activeTab) {
            api.instances.setParent(name, activeTab, detail.selectedInstance.id, { parent_type: '', parent_instance_id: '' })
              .then(() => { setSetParentOpen(false); inst.loadInstances(); detail.selectInstance(detail.selectedInstance) })
              .catch(() => {})
          }
        }}
        error={setParentError}
      />

      {/* Delete Instance Modal */}
      <Modal variant={ModalVariant.small} isOpen={inst.deleteTarget !== null} onClose={inst.closeDelete}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {inst.deleteError && <Alert variant="danger" title={inst.deleteError} isInline style={{ marginBottom: '1rem' }} />}
          Are you sure you want to delete <strong>{inst.deleteTarget?.name}</strong>? Contained instances will also be deleted.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={inst.handleDelete}>Delete</Button>
          <Button variant="link" onClick={inst.closeDelete}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Copy Catalog Modal */}
      <CopyCatalogModal
        isOpen={copyOpen}
        onClose={() => { setCopyOpen(false); setCopyError(null) }}
        onSubmit={handleCopy}
        error={copyError}
        loading={copyLoading}
      />

      {/* Replace Catalog Modal */}
      <ReplaceCatalogModal
        isOpen={replaceOpen}
        onClose={() => { setReplaceOpen(false); setReplaceError(null) }}
        onSubmit={handleReplace}
        availableCatalogs={availableCatalogs}
        error={replaceError}
        loading={replaceLoading}
      />
    </PageSection>
  )
}
