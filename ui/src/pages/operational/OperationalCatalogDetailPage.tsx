import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Tabs,
  Tab,
  TabTitleText,
  Button,
  Label,
  EmptyState,
  EmptyStateBody,
  Spinner,
  Breadcrumb,
  BreadcrumbItem,
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Form,
  FormGroup,
  FormSelect,
  FormSelectOption,
  Alert,
} from '@patternfly/react-core'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, CatalogVersionPin, TreeNodeResponse, SnapshotAttribute, SnapshotAssociation, Role } from '../../types'
import type { AddChildCreateData, AddChildAdoptData } from '../../components/AddChildModal'
import AttributeFormFields, { validateInstanceName } from '../../components/AttributeFormFields'
import AddChildModal from '../../components/AddChildModal'
import SetParentModal from '../../components/SetParentModal'
import LinkModal from '../../components/LinkModal'
import { buildTypedAttrs } from '../../utils/buildTypedAttrs'
import { errorMessage } from '../../utils/errorMessage'
import { loadSchemaSnapshot } from '../../utils/loadSchemaSnapshot'
import { useValidation } from '../../hooks/useValidation'
import { useContainmentTree } from '../../hooks/useContainmentTree'
import { useCatalogDiagram } from '../../hooks/useCatalogDiagram'
import ValidationResults from '../../components/ValidationResults'
import InstanceDetailPanel from '../../components/InstanceDetailPanel'
import DiagramTabContent from '../../components/DiagramTabContent'

export default function OperationalCatalogDetailPage({ role }: { role: Role }) {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()

  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string>('tree')
  const [pins, setPins] = useState<CatalogVersionPin[]>([])

  const ct = useContainmentTree(name)

  const loadCatalog = useCallback(async () => {
    if (!name) return
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const cat = await api.catalogs.get(name)
      setCatalog(cat)
      if (cat?.catalog_version_id) {
        try {
          const pinsRes = await api.catalogVersions.listPins(cat.catalog_version_id)
          setPins(pinsRes.items || [])
        } catch { /* pins loading is non-fatal */ }
      }
    } catch (e) {
      setError(errorMessage(e, 'Failed to load catalog'))
    } finally {
      setLoading(false)
    }
  }, [name, role])

  useEffect(() => { loadCatalog() }, [loadCatalog])

  const canWrite = role === 'RW' || role === 'Admin' || role === 'SuperAdmin'
  const canMutate = canWrite && (!catalog?.published || role === 'SuperAdmin')

  // Create instance modal state
  const [createOpen, setCreateOpen] = useState(false)
  const [createType, setCreateType] = useState('')
  const [createSchemaAttrs, setCreateSchemaAttrs] = useState<SnapshotAttribute[]>([])
  const [createEnumValues, setCreateEnumValues] = useState<Record<string, string[]>>({})
  const [createName, setCreateName] = useState('')
  const [createDesc, setCreateDesc] = useState('')
  const [createAttrs, setCreateAttrs] = useState<Record<string, string>>({})
  const [createError, setCreateError] = useState<string | null>(null)

  // Edit instance modal state
  const [editOpen, setEditOpen] = useState(false)
  const [editSchemaAttrs, setEditSchemaAttrs] = useState<SnapshotAttribute[]>([])
  const [editEnumValues, setEditEnumValues] = useState<Record<string, string[]>>({})
  const [editName, setEditName] = useState('')
  const [editDesc, setEditDesc] = useState('')
  const [editAttrs, setEditAttrs] = useState<Record<string, string>>({})
  const [editError, setEditError] = useState<string | null>(null)

  // Selected instance schema (loaded when instance selected, for containment/link buttons)
  const [selectedSchemaAssocs, setSelectedSchemaAssocs] = useState<SnapshotAssociation[]>([])

  // Inline action error (shown in detail panel, not page-level)
  const [actionError, setActionError] = useState<string | null>(null)

  // Delete instance modal state
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleteChildren, setDeleteChildren] = useState<TreeNodeResponse[]>([])
  const [deleteParentId, setDeleteParentId] = useState<string | null>(null)

  // Add child modal state
  const [addChildOpen, setAddChildOpen] = useState(false)
  const [addChildError, setAddChildError] = useState<string | null>(null)

  // Mutation submitting state (prevents double-clicks)
  const [submitting, setSubmitting] = useState(false)

  // Set parent modal state
  const [setParentOpen, setSetParentOpen] = useState(false)
  const [parentTypeName, setParentTypeName] = useState('')
  const [setParentError, setSetParentError] = useState<string | null>(null)

  // Link modal state
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkError, setLinkError] = useState<string | null>(null)

  useEffect(() => {
    setActionError(null)
  }, [ct.selectedNodeId])

  useEffect(() => {
    if (!ct.selectedTypeName) { setSelectedSchemaAssocs([]); return }
    loadSchemaSnapshot(pins, ct.selectedTypeName).then(({ assocs }) => setSelectedSchemaAssocs(assocs))
  }, [ct.selectedTypeName, pins])

  const handleCreateTypeChange = useCallback(async (typeName: string) => {
    setCreateType(typeName)
    if (!typeName) return
    const { attrs, enums } = await loadSchemaSnapshot(pins, typeName)
    setCreateSchemaAttrs(attrs)
    setCreateEnumValues(enums)
    const initial: Record<string, string> = {}
    for (const attr of attrs) {
      if (!attr.system && attr.base_type === 'boolean' && attr.required) initial[attr.name] = 'false'
    }
    setCreateAttrs(initial)
  }, [pins])

  const openCreateModal = useCallback((preselectedType?: string) => {
    setCreateOpen(true)
    setCreateError(null)
    setCreateName('')
    setCreateDesc('')
    setCreateAttrs({})
    if (preselectedType) {
      handleCreateTypeChange(preselectedType)
    } else {
      setCreateType('')
      setCreateSchemaAttrs([])
      setCreateEnumValues({})
    }
  }, [handleCreateTypeChange])

  const handleCreateSubmit = useCallback(async () => {
    if (!name || !createType || !createName.trim() || submitting) return
    setCreateError(null)
    setSubmitting(true)
    try {
      const typed = buildTypedAttrs(createAttrs, createSchemaAttrs)
      await api.instances.create(name, createType, {
        name: createName.trim(),
        description: createDesc.trim() || undefined,
        ...(Object.keys(typed).length > 0 ? { attributes: typed } : {}),
      })
      setCreateOpen(false)
      await ct.loadTree()
    } catch (e) {
      setCreateError(errorMessage(e, 'Failed to create instance'))
    } finally {
      setSubmitting(false)
    }
  }, [name, createType, createName, createDesc, createAttrs, createSchemaAttrs, ct, submitting])

  const openEditModal = useCallback(async () => {
    if (!ct.selectedInstance || !ct.selectedTypeName) return
    setEditError(null)
    const { attrs, enums } = await loadSchemaSnapshot(pins, ct.selectedTypeName)
    setEditSchemaAttrs(attrs)
    setEditEnumValues(enums)
    setEditName(ct.selectedInstance.name)
    setEditDesc(ct.selectedInstance.description || '')
    const vals: Record<string, string> = {}
    for (const a of ct.selectedInstance.attributes || []) {
      if (a.system) continue
      vals[a.name] = a.value != null ? String(a.value) : ''
    }
    setEditAttrs(vals)
    setEditOpen(true)
  }, [ct.selectedInstance, ct.selectedTypeName, pins])

  const handleEditSubmit = useCallback(async () => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || !editName.trim() || submitting) return
    setEditError(null)
    setSubmitting(true)
    try {
      const typed = buildTypedAttrs(editAttrs, editSchemaAttrs, true)
      await api.instances.update(name, ct.selectedTypeName, ct.selectedInstance.id, {
        version: ct.selectedInstance.version,
        name: editName.trim(),
        description: editDesc.trim(),
        ...(Object.keys(typed).length > 0 ? { attributes: typed } : {}),
      })
      setEditOpen(false)
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setEditError(errorMessage(e, 'Failed to update instance'))
    } finally {
      setSubmitting(false)
    }
  }, [name, ct, editName, editDesc, editAttrs, editSchemaAttrs, submitting])

  const openDeleteModal = useCallback(() => {
    if (!ct.selectedInstance) return
    setDeleteError(null)
    setDeleteChildren(ct.getDescendants(ct.selectedInstance.id, ct.tree))
    const parentNode = ct.findParentNode(ct.selectedInstance.id, ct.tree)
    setDeleteParentId(parentNode?.instance_id ?? null)
    setDeleteOpen(true)
  }, [ct])

  const handleDeleteSubmit = useCallback(async () => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || submitting) return
    setDeleteError(null)
    setSubmitting(true)
    try {
      await api.instances.delete(name, ct.selectedTypeName, ct.selectedInstance.id)
      setDeleteOpen(false)
      const freshTree = await ct.loadTree()
      if (deleteParentId) {
        await ct.selectNodeById(deleteParentId, freshTree)
      } else {
        ct.clearSelection()
      }
    } catch (e) {
      setDeleteError(errorMessage(e, 'Failed to delete instance'))
    } finally {
      setSubmitting(false)
    }
  }, [name, ct, submitting, deleteParentId])

  const outgoingContainment = selectedSchemaAssocs.filter(a => a.type === 'containment' && a.direction === 'outgoing')
  const incomingContainment = selectedSchemaAssocs.filter(a => a.type === 'containment' && a.direction === 'incoming')
  const outgoingLinks = selectedSchemaAssocs.filter(a => a.type !== 'containment' &&
    (a.direction === 'outgoing' || (a.direction === 'incoming' && a.type === 'bidirectional')))

  const handleAddChild = useCallback(async (childType: string, mode: 'create' | 'adopt', data: AddChildCreateData | AddChildAdoptData) => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || !childType) return
    setAddChildError(null)
    try {
      if (mode === 'adopt') {
        const adoptData = data as AddChildAdoptData
        await api.instances.setParent(name, childType, adoptData.adoptInstanceId, {
          parent_type: ct.selectedTypeName,
          parent_instance_id: ct.selectedInstance.id,
        })
      } else {
        const createData = data as AddChildCreateData
        if (!createData.name.trim()) return
        await api.instances.createContained(name, ct.selectedTypeName, ct.selectedInstance.id, childType, {
          name: createData.name.trim(),
          description: createData.description.trim() || undefined,
          ...(Object.keys(createData.attrs).length > 0 ? { attributes: createData.attrs } : {}),
        })
      }
      setAddChildOpen(false)
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setAddChildError(errorMessage(e, 'Failed to add child'))
    }
  }, [name, ct])

  const handleSetParent = useCallback(async (pType: string, pId: string) => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || !pType) return
    setSetParentError(null)
    try {
      await api.instances.setParent(name, ct.selectedTypeName, ct.selectedInstance.id, {
        parent_type: pType,
        parent_instance_id: pId,
      })
      setSetParentOpen(false)
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setSetParentError(errorMessage(e, 'Failed to set parent'))
    }
  }, [name, ct])

  const handleRemoveFromContainer = useCallback(async () => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || submitting) return
    setSubmitting(true)
    setActionError(null)
    try {
      await api.instances.setParent(name, ct.selectedTypeName, ct.selectedInstance.id, {
        parent_type: '',
        parent_instance_id: '',
      })
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setActionError(errorMessage(e, 'Failed to remove from container'))
    } finally {
      setSubmitting(false)
    }
  }, [name, ct, submitting])

  const handleCreateLink = useCallback(async (targetId: string, assocName: string) => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || !targetId || !assocName) return
    setLinkError(null)
    try {
      await api.links.create(name, ct.selectedTypeName, ct.selectedInstance.id, {
        target_instance_id: targetId,
        association_name: assocName,
      })
      setLinkOpen(false)
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setLinkError(errorMessage(e, 'Failed to create link'))
    }
  }, [name, ct])

  const handleUnlink = useCallback(async (linkId: string) => {
    if (!name || !ct.selectedInstance || !ct.selectedTypeName || submitting) return
    setSubmitting(true)
    setActionError(null)
    try {
      await api.links.delete(name, ct.selectedTypeName, ct.selectedInstance.id, linkId)
      const freshTree = await ct.loadTree()
      await ct.selectNodeById(ct.selectedInstance.id, freshTree)
    } catch (e) {
      setActionError(errorMessage(e, 'Failed to unlink'))
    } finally {
      setSubmitting(false)
    }
  }, [name, ct, submitting])

  const validation = useValidation(name, loadCatalog)
  const diagram = useCatalogDiagram(catalog?.catalog_version_id)

  useEffect(() => {
    if (activeTab === 'tree') ct.loadTree()
    if (activeTab === '__diagram__') diagram.loadDiagram()
  }, [activeTab, ct.loadTree, diagram.loadDiagram])

  const renderTreeNode = (node: TreeNodeResponse, depth: number) => {
    const hasChildren = node.children && node.children.length > 0
    const isExpanded = ct.expandedNodes.has(node.instance_id)
    const isSelected = ct.selectedNodeId === node.instance_id

    return (
      <div key={node.instance_id}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '4px 8px',
            paddingLeft: `${depth * 20 + 8}px`,
            cursor: 'pointer',
            backgroundColor: isSelected ? 'var(--pf-t--global--color--brand--default)' : 'transparent',
            color: isSelected ? 'var(--pf-t--global--text--color--on-brand--default, #fff)' : 'inherit',
            borderRadius: '3px',
            marginBottom: '1px',
          }}
          onClick={() => ct.selectTreeNode(node)}
        >
          {hasChildren && (
            <span
              style={{ marginRight: '6px', cursor: 'pointer', userSelect: 'none', width: '16px', display: 'inline-block' }}
              onClick={(e) => { e.stopPropagation(); ct.toggleNode(node.instance_id) }}
            >
              {isExpanded ? '▾' : '▸'}
            </span>
          )}
          {!hasChildren && <span style={{ width: '22px', display: 'inline-block' }} />}
          <span style={{ fontWeight: 500 }}>{node.instance_name}</span>
        </div>
        {hasChildren && isExpanded && node.children.map(child => renderTreeNode(child, depth + 1))}
      </div>
    )
  }

  const groupedTree = useMemo(() => {
    const groups: Record<string, TreeNodeResponse[]> = {}
    for (const node of ct.tree) {
      if (!groups[node.entity_type_name]) groups[node.entity_type_name] = []
      groups[node.entity_type_name].push(node)
    }
    return groups
  }, [ct.tree])

  const renderEntityTypeGroup = (typeName: string, nodes: TreeNodeResponse[]) => {
    const groupKey = `__group__${typeName}`
    const isExpanded = ct.expandedNodes.has(groupKey)

    return (
      <div key={groupKey}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '4px 8px',
            cursor: 'pointer',
            fontWeight: 600,
            borderRadius: '3px',
            marginBottom: '1px',
          }}
          onClick={() => ct.toggleNode(groupKey)}
        >
          <span style={{ marginRight: '6px', userSelect: 'none', width: '16px', display: 'inline-block' }}>
            {isExpanded ? '▾' : '▸'}
          </span>
          <span style={{ flex: 1 }}>{typeName} ({nodes.length})</span>
          {canMutate && (
            <Button variant="plain" size="sm" style={{ padding: '0 4px', minWidth: 'auto' }}
              onClick={(e) => { e.stopPropagation(); openCreateModal(typeName) }}
              aria-label={`Create ${typeName}`}>+</Button>
          )}
        </div>
        {isExpanded && nodes.map(node => renderTreeNode(node, 1))}
      </div>
    )
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !catalog) return <PageSection><div style={{ color: '#c9190b' }}>{error}</div></PageSection>
  if (!catalog) return <PageSection><div>Catalog not found</div></PageSection>

  return (
    <PageSection>
      {error && catalog && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}
      <Breadcrumb style={{ marginBottom: '1rem' }}>
        <BreadcrumbItem>
          <Button variant="link" isInline onClick={() => navigate('/')}>Catalogs</Button>
        </BreadcrumbItem>
        <BreadcrumbItem isActive>{catalog.name}</BreadcrumbItem>
      </Breadcrumb>

      <Title headingLevel="h1">
        {catalog.name}{' '}
        <Label color={catalog.validation_status === 'valid' ? 'green' : catalog.validation_status === 'invalid' ? 'red' : 'blue'}>
          {catalog.validation_status}
        </Label>
        {catalog.published && (
          <Label color="purple" style={{ marginLeft: '0.5rem' }}>published</Label>
        )}
      </Title>
      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>
        Catalog Version: {catalog.catalog_version_label || catalog.catalog_version_id}
        {catalog.description && ` — ${catalog.description}`}
      </p>

      {canWrite && (
        <div style={{ marginBottom: '1rem' }}>
          <Button variant="secondary" onClick={validation.validate} isLoading={validation.validating} isDisabled={validation.validating}>
            Validate
          </Button>
        </div>
      )}

      <ValidationResults errors={validation.errors} ran={validation.ran} error={validation.error} />

      <Tabs activeKey={activeTab} onSelect={(_e, key) => setActiveTab(String(key))} style={{ marginTop: '1rem' }}>
        <Tab eventKey="tree" title={<TabTitleText>Tree Browser</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <div style={{ display: 'flex', gap: '1rem' }}>
              {/* Tree panel (left) */}
              <div style={{ width: '300px', minWidth: '250px', borderRight: '1px solid #d2d2d2', paddingRight: '1rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                  <Title headingLevel="h4">Containment Tree</Title>
                  {canMutate && (
                    <Button variant="primary" size="sm" onClick={() => openCreateModal()}>
                      Create Instance
                    </Button>
                  )}
                </div>
                {ct.treeLoading ? (
                  <Spinner size="md" aria-label="Loading tree" />
                ) : ct.tree.length === 0 ? (
                  <p style={{ color: '#6a6e73' }}>No instances in this catalog.</p>
                ) : (
                  <div style={{ maxHeight: '600px', overflow: 'auto' }}>
                    {Object.entries(groupedTree).map(([typeName, nodes]) =>
                      renderEntityTypeGroup(typeName, nodes)
                    )}
                  </div>
                )}
              </div>

              {/* Detail panel (right) */}
              <div style={{ flex: 1 }}>
                {ct.detailLoading ? (
                  <Spinner aria-label="Loading detail" />
                ) : ct.selectedInstance ? (
                  <>
                    {canMutate && (
                      <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
                        <Button variant="secondary" size="sm" onClick={openEditModal}>Edit</Button>
                        <Button variant="danger" size="sm" onClick={openDeleteModal}>Delete</Button>
                        {outgoingContainment.length > 0 && (
                          <Button variant="secondary" size="sm" onClick={() => { setAddChildError(null); setAddChildOpen(true) }}>Add Child</Button>
                        )}
                        {incomingContainment.length > 0 && (
                          <Button variant="secondary" size="sm" onClick={() => {
                            setSetParentError(null)
                            const inc = incomingContainment[0]
                            if (inc?.source_entity_type_name) setParentTypeName(inc.source_entity_type_name)
                            setSetParentOpen(true)
                          }}>Set Parent</Button>
                        )}
                        {ct.selectedInstance?.parent_instance_id && (
                          <Button variant="warning" size="sm" onClick={handleRemoveFromContainer} isDisabled={submitting}>Remove from Container</Button>
                        )}
                        {outgoingLinks.length > 0 && (
                          <Button variant="secondary" size="sm" onClick={() => { setLinkError(null); setLinkOpen(true) }}>Create Link</Button>
                        )}
                      </div>
                    )}
                    <InstanceDetailPanel
                      instance={ct.selectedInstance}
                      catalogName={catalog.name}
                      forwardRefs={ct.forwardRefs}
                      reverseRefs={ct.reverseRefs}
                      refsLoading={ct.refsLoading}
                      onNavigateToRef={ct.navigateToTreeNode}
                      onUnlink={canMutate ? handleUnlink : undefined}
                      unlinkDisabled={submitting}
                      actionError={actionError}
                    />
                  </>
                ) : (
                  <EmptyState>
                    <EmptyStateBody>
                      Select an instance from the tree to view its details.
                    </EmptyStateBody>
                  </EmptyState>
                )}
              </div>
            </div>
          </PageSection>
        </Tab>

        <Tab eventKey="__diagram__" title={<TabTitleText>Model Diagram</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <DiagramTabContent
              diagramData={diagram.diagramData}
              diagramLoading={diagram.diagramLoading}
              diagramError={diagram.diagramError}
            />
          </PageSection>
        </Tab>
      </Tabs>

      {/* Create Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={createOpen} onClose={() => setCreateOpen(false)}>
        <ModalHeader title="Create Instance" />
        <ModalBody>
          {createError && <Alert variant="danger" title={createError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Entity Type" isRequired fieldId="create-entity-type">
              <FormSelect
                id="create-entity-type"
                value={createType}
                onChange={(_e, val) => handleCreateTypeChange(val)}
                aria-label="Entity type"
              >
                <FormSelectOption value="" label="Select entity type..." isDisabled />
                {pins.map(p => (
                  <FormSelectOption key={p.entity_type_name} value={p.entity_type_name} label={p.entity_type_name} />
                ))}
              </FormSelect>
            </FormGroup>
            {createType && (
              <AttributeFormFields
                schemaAttrs={createSchemaAttrs}
                values={createAttrs}
                onChange={(n, v) => setCreateAttrs(prev => ({ ...prev, [n]: v }))}
                enumValues={createEnumValues}
                idPrefix="create-inst"
                includeSystem
                systemName={createName}
                setSystemName={setCreateName}
                systemDesc={createDesc}
                setSystemDesc={setCreateDesc}
              />
            )}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreateSubmit} isDisabled={!createType || !createName.trim() || !!validateInstanceName(createName) || submitting} isLoading={submitting}>Create</Button>
          <Button variant="link" onClick={() => setCreateOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Edit Instance Modal */}
      <Modal variant={ModalVariant.medium} isOpen={editOpen} onClose={() => setEditOpen(false)}>
        <ModalHeader title={`Edit ${ct.selectedInstance?.name || ''}`} />
        <ModalBody>
          {editError && <Alert variant="danger" title={editError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <AttributeFormFields
              schemaAttrs={editSchemaAttrs}
              values={editAttrs}
              onChange={(n, v) => setEditAttrs(prev => ({ ...prev, [n]: v }))}
              enumValues={editEnumValues}
              idPrefix="edit-inst"
              includeSystem
              systemName={editName}
              setSystemName={setEditName}
              systemDesc={editDesc}
              setSystemDesc={setEditDesc}
            />
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleEditSubmit} isDisabled={!editName.trim() || !!validateInstanceName(editName) || submitting} isLoading={submitting}>Save</Button>
          <Button variant="link" onClick={() => setEditOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Instance Modal */}
      <Modal variant={ModalVariant.small} isOpen={deleteOpen} onClose={() => setDeleteOpen(false)}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          {deleteError && <Alert variant="danger" title={deleteError} isInline style={{ marginBottom: '1rem' }} />}
          <p>Are you sure you want to delete <strong>{ct.selectedInstance?.name}</strong>?</p>
          {deleteChildren.length > 0 && (
            <Alert variant="warning" title={`${deleteChildren.length} contained instance(s) will also be deleted:`} isInline style={{ marginTop: '0.5rem' }}>
              <ul>
                {deleteChildren.map(c => <li key={c.instance_id}>{c.instance_name}</li>)}
              </ul>
            </Alert>
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDeleteSubmit} isDisabled={submitting} isLoading={submitting}>Delete</Button>
          <Button variant="link" onClick={() => setDeleteOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Add Child Modal */}
      <AddChildModal
        isOpen={addChildOpen}
        onClose={() => { setAddChildOpen(false); setAddChildError(null) }}
        catalogName={name}
        pins={pins}
        schemaAssocs={selectedSchemaAssocs}
        onSubmit={handleAddChild}
        error={addChildError}
      />

      {/* Set Parent Modal */}
      <SetParentModal
        isOpen={setParentOpen}
        onClose={() => { setSetParentOpen(false); setSetParentError(null) }}
        catalogName={name}
        instanceName={ct.selectedInstance?.name}
        parentTypeName={parentTypeName}
        hasParent={!!ct.selectedInstance?.parent_instance_id}
        onSubmit={handleSetParent}
        onRemoveParent={handleRemoveFromContainer}
        error={setParentError}
      />

      {/* Link Modal */}
      <LinkModal
        isOpen={linkOpen}
        onClose={() => { setLinkOpen(false); setLinkError(null) }}
        catalogName={name}
        pins={pins}
        schemaAssocs={selectedSchemaAssocs}
        onSubmit={handleCreateLink}
        error={linkError}
        instanceNames={ct.instanceNames}
      />
    </PageSection>
  )
}
