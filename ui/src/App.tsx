import '@patternfly/patternfly/patternfly.css'
import { useEffect, useState, useCallback } from 'react'
import { Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import {
  Page,
  Masthead,
  MastheadMain,
  MastheadBrand,
  MastheadContent,
  PageSection,
  Title,
  Toolbar,
  ToolbarItem,
  ToolbarContent,
  SearchInput,
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
  Tabs,
  Tab,
  TabTitleText,
  Label,
  EmptyState,
  EmptyStateBody,
  Spinner,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from './api/client'
import type { EntityType, CatalogVersion, ContainmentTreeNode, Role, VersionSnapshot } from './types'
import EntityTypeDiagram, { type DiagramEntityType, type EdgeClickData } from './components/EntityTypeDiagram'
import EditAssociationModal from './components/EditAssociationModal'
import EntityTypeDetailPage from './pages/meta/EntityTypeDetailPage'
import TypeDefinitionListPage from './pages/meta/TypeDefinitionListPage'
import TypeDefinitionDetailPage from './pages/meta/TypeDefinitionDetailPage'
import CatalogVersionDetailPage from './pages/meta/CatalogVersionDetailPage'
import CatalogListPage from './pages/meta/CatalogListPage'
import CatalogDetailPage from './pages/meta/CatalogDetailPage'
import OperationalCatalogDetailPage from './pages/operational/OperationalCatalogDetailPage'
import LandingPage from './pages/LandingPage'

const ROLES: Role[] = ['RO', 'RW', 'Admin', 'SuperAdmin']

function TreeNodeRow({
  node,
  depth,
  selectedVersions,
  onToggle,
  onVersionChange,
}: {
  node: ContainmentTreeNode
  depth: number
  selectedVersions: Record<string, string>
  onToggle: (entityTypeId: string, checked: boolean) => void
  onVersionChange: (entityTypeId: string, versionId: string) => void
}) {
  const isChecked = !!selectedVersions[node.entity_type.id]
  const latestVersionId = node.versions.find(v => v.version === node.latest_version)?.id || node.versions[0]?.id || ''
  const selectedVersionId = selectedVersions[node.entity_type.id] || latestVersionId

  return (
    <>
      <div style={{ display: 'flex', alignItems: 'center', padding: '0.35rem 0', marginLeft: `${depth * 1.5}rem` }}>
        <input
          type="checkbox"
          aria-label={node.entity_type.name}
          checked={isChecked}
          onChange={(e) => onToggle(node.entity_type.id, e.target.checked)}
          style={{ marginRight: '0.5rem' }}
        />
        <span style={{ flex: 1, fontWeight: node.children.length > 0 ? 600 : 400 }}>
          {node.entity_type.name}
        </span>
        <select
          aria-label={`Version for ${node.entity_type.name}`}
          value={selectedVersionId}
          onChange={(e) => onVersionChange(node.entity_type.id, e.target.value)}
          style={{ marginLeft: '0.5rem', minWidth: '4rem' }}
        >
          {node.versions.map((v) => (
            <option key={v.id} value={v.id}>V{v.version}</option>
          ))}
        </select>
      </div>
      {node.children.map((child) => (
        <TreeNodeRow
          key={child.entity_type.id}
          node={child}
          depth={depth + 1}
          selectedVersions={selectedVersions}
          onToggle={onToggle}
          onVersionChange={onVersionChange}
        />
      ))}
    </>
  )
}

function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const [role, setRole] = useState<Role>('Admin')
  const [roleSelectOpen, setRoleSelectOpen] = useState(false)

  // Determine active tab from URL (schema routes have /schema prefix)
  const getActiveTab = () => {
    if (!location.pathname.startsWith('/schema')) return ''
    if (location.pathname.startsWith('/schema/catalog-versions')) return 'catalogVersions'
    if (location.pathname.startsWith('/schema/catalogs')) return 'catalogs'
    if (location.pathname.startsWith('/schema/types')) return 'types'
    if (location.pathname.startsWith('/schema/model-diagram')) return 'modelDiagram'
    return 'entityTypes'
  }

  // Context-aware masthead title
  const getMastheadTitle = () => {
    if (location.pathname.startsWith('/schema')) return 'AI Asset Hub — Schema'
    if (location.pathname.startsWith('/catalogs')) return 'AI Asset Hub — Data Viewer'
    return 'AI Asset Hub'
  }

  // Entity Types state
  const [entityTypes, setEntityTypes] = useState<EntityType[]>([])
  const [etTotal, setEtTotal] = useState(0)
  const [etFilter, setEtFilter] = useState('')
  const [etLoading, setEtLoading] = useState(false)
  const [etError, setEtError] = useState<string | null>(null)

  // Catalog Versions state
  const [catalogVersions, setCatalogVersions] = useState<CatalogVersion[]>([])
  const [cvTotal, setCvTotal] = useState(0)
  const [cvLoading, setCvLoading] = useState(false)
  const [cvError, setCvError] = useState<string | null>(null)

  // Create Entity Type modal
  const [createEtOpen, setCreateEtOpen] = useState(false)
  const [newEtName, setNewEtName] = useState('')
  const [newEtDesc, setNewEtDesc] = useState('')
  const [createEtError, setCreateEtError] = useState<string | null>(null)

  // Delete confirmation modal
  const [deleteTarget, setDeleteTarget] = useState<EntityType | null>(null)

  // Delete catalog version confirmation
  const [deleteCvTarget, setDeleteCvTarget] = useState<CatalogVersion | null>(null)

  // Stage filter
  const [stageFilter, setStageFilter] = useState('')
  const [stageFilterOpen, setStageFilterOpen] = useState(false)

  // Create Catalog Version modal
  const [createCvOpen, setCreateCvOpen] = useState(false)
  const [newCvLabel, setNewCvLabel] = useState('')
  const [newCvDesc, setNewCvDesc] = useState('')
  const [createCvError, setCreateCvError] = useState<string | null>(null)
  const [cvTree, setCvTree] = useState<ContainmentTreeNode[]>([])
  const [cvSelectedVersions, setCvSelectedVersions] = useState<Record<string, string>>({})
  const [cvEtLoading, setCvEtLoading] = useState(false)

  // Sync role to API client
  useEffect(() => {
    setAuthRole(role)
  }, [role])

  // Fetch entity types
  const loadEntityTypes = useCallback(async () => {
    setEtLoading(true)
    setEtError(null)
    try {
      const res = await api.entityTypes.list()
      setEntityTypes(res.items || [])
      setEtTotal(res.total)
    } catch (e) {
      setEtError(e instanceof Error ? e.message : 'Failed to load entity types')
    } finally {
      setEtLoading(false)
    }
  }, [])

  // Fetch catalog versions
  const loadCatalogVersions = useCallback(async () => {
    setCvLoading(true)
    setCvError(null)
    try {
      const res = await api.catalogVersions.list(stageFilter ? { stage: stageFilter } : undefined)
      setCatalogVersions(res.items || [])
      setCvTotal(res.total)
    } catch (e) {
      setCvError(e instanceof Error ? e.message : 'Failed to load catalog versions')
    } finally {
      setCvLoading(false)
    }
  }, [stageFilter])

  // Load data on mount, tab change, and route change (e.g., navigating back from detail page)
  const activeTab = getActiveTab()
  useEffect(() => {
    setAuthRole(role)
    if (activeTab === 'entityTypes' && (location.pathname === '/schema' || location.pathname === '/schema/')) loadEntityTypes()
    if (activeTab === 'catalogVersions' && location.pathname === '/schema/catalog-versions') loadCatalogVersions()
  }, [activeTab, role, location.pathname, loadEntityTypes, loadCatalogVersions])

  // Model diagram state
  const [diagramData, setDiagramData] = useState<DiagramEntityType[]>([])
  const [diagramLoading, setDiagramLoading] = useState(false)

  // Diagram edit association modal state
  const [diagramEditOpen, setDiagramEditOpen] = useState(false)
  const [diagramEditSourceId, setDiagramEditSourceId] = useState('')
  const [diagramEditData, setDiagramEditData] = useState({
    name: '', type: '', sourceRole: '', targetRole: '',
    sourceCardinality: '0..n', targetCardinality: '0..n',
    sourceName: '', targetName: '',
  })

  const handleDiagramEdgeClick = (edgeData: EdgeClickData) => {
    setDiagramEditSourceId(edgeData.sourceEntityTypeId)
    setDiagramEditData({
      name: edgeData.name,
      type: edgeData.assocType,
      sourceRole: edgeData.sourceRole,
      targetRole: edgeData.targetRole,
      sourceCardinality: edgeData.sourceCardinality || '0..n',
      targetCardinality: edgeData.targetCardinality || '0..n',
      sourceName: edgeData.sourceEntityTypeName,
      targetName: edgeData.targetEntityTypeName,
    })
    setDiagramEditOpen(true)
  }

  const handleDiagramEditSave = async (data: { name: string; type: string; sourceRole: string; targetRole: string; sourceCardinality: string; targetCardinality: string }) => {
    if (!diagramEditSourceId) return
    const req: Record<string, string | undefined> = {}
    if (data.name !== diagramEditData.name) req.name = data.name
    req.type = data.type
    req.source_role = data.sourceRole
    req.target_role = data.targetRole
    req.source_cardinality = data.sourceCardinality
    req.target_cardinality = data.targetCardinality
    await api.associations.edit(diagramEditSourceId, diagramEditData.name, req)
    setDiagramEditOpen(false)
    loadDiagramData()
  }

  const loadDiagramData = useCallback(async () => {
    setDiagramLoading(true)
    try {
      const etResult = await api.entityTypes.list()
      const items = etResult.items || []
      const snapshots: DiagramEntityType[] = await Promise.all(
        items.map(async (et: EntityType) => {
          const versions = await api.versions.list(et.id)
          const latest = versions.items?.length ? Math.max(...versions.items.map((v: any) => v.version)) : 1
          const snapshot: VersionSnapshot = await api.versions.snapshot(et.id, latest)
          return {
            entityType: et,
            version: latest,
            attributes: snapshot.attributes || [],
            associations: snapshot.associations || [],
          }
        })
      )
      setDiagramData(snapshots)
    } catch {
      // Diagram data loading failed — show empty diagram
    } finally {
      setDiagramLoading(false)
    }
  }, [])

  useEffect(() => {
    if (activeTab === 'modelDiagram' && location.pathname === '/schema/model-diagram') loadDiagramData()
  }, [activeTab, location.pathname, loadDiagramData])

  // Create entity type
  const handleCreateEntityType = async () => {
    if (!newEtName.trim()) return
    setCreateEtError(null)
    try {
      await api.entityTypes.create({ name: newEtName.trim(), description: newEtDesc.trim() || undefined })
      setCreateEtOpen(false)
      setNewEtName('')
      setNewEtDesc('')
      loadEntityTypes()
    } catch (e) {
      setCreateEtError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  // Delete entity type (with confirmation)
  const handleDeleteEntityType = async () => {
    if (!deleteTarget) return
    try {
      await api.entityTypes.delete(deleteTarget.id)
      setDeleteTarget(null)
      loadEntityTypes()
    } catch (e) {
      setEtError(e instanceof Error ? e.message : 'Failed to delete')
      setDeleteTarget(null)
    }
  }

  // Load containment tree for CV creation
  const loadCvTree = async () => {
    setCvEtLoading(true)
    try {
      const tree = await api.entityTypes.containmentTree()
      setCvTree(tree || [])
    } catch { /* ignore */ }
    setCvEtLoading(false)
  }

  // Collect all entity type IDs from tree nodes recursively
  const collectDescendantIds = (nodes: ContainmentTreeNode[]): string[] => {
    const ids: string[] = []
    for (const node of nodes) {
      ids.push(node.entity_type.id)
      ids.push(...collectDescendantIds(node.children))
    }
    return ids
  }

  // Collect all ancestor IDs for a given entity type ID in the tree
  const collectAncestorIds = (nodes: ContainmentTreeNode[], targetId: string, path: string[] = []): string[] | null => {
    for (const node of nodes) {
      if (node.entity_type.id === targetId) return path
      const result = collectAncestorIds(node.children, targetId, [...path, node.entity_type.id])
      if (result !== null) return result
    }
    return null
  }

  // Find a node by ID in the tree
  const findNode = (nodes: ContainmentTreeNode[], id: string): ContainmentTreeNode | null => {
    for (const node of nodes) {
      if (node.entity_type.id === id) return node
      const found = findNode(node.children, id)
      if (found) return found
    }
    return null
  }

  // Get the latest version ID for a given entity type node
  const getLatestVersionId = (node: ContainmentTreeNode): string => {
    const latest = node.versions.find(v => v.version === node.latest_version)
    return latest?.id || node.versions[0]?.id || ''
  }

  // Handle checkbox toggle with cascade logic
  const handleTreeSelect = (entityTypeId: string, checked: boolean) => {
    setCvSelectedVersions(prev => {
      const next = { ...prev }
      if (checked) {
        // Select this node with its latest version
        const node = findNode(cvTree, entityTypeId)
        if (node) {
          next[entityTypeId] = getLatestVersionId(node)
          // Auto-select all descendants recursively
          const descendantIds = collectDescendantIds(node.children)
          for (const did of descendantIds) {
            const dNode = findNode(cvTree, did)
            if (dNode && !next[did]) {
              next[did] = getLatestVersionId(dNode)
            }
          }
          // Auto-select all ancestors up to root
          const ancestorIds = collectAncestorIds(cvTree, entityTypeId) || []
          for (const aid of ancestorIds) {
            const aNode = findNode(cvTree, aid)
            if (aNode && !next[aid]) {
              next[aid] = getLatestVersionId(aNode)
            }
          }
        }
      } else {
        // Deselect this node
        delete next[entityTypeId]
        // Deselect all descendants recursively
        const node = findNode(cvTree, entityTypeId)
        if (node) {
          const descendantIds = collectDescendantIds(node.children)
          for (const did of descendantIds) {
            delete next[did]
          }
        }
        // Do NOT deselect ancestors
      }
      return next
    })
  }

  // Create catalog version
  const handleCreateCatalogVersion = async () => {
    if (!newCvLabel.trim()) return
    setCreateCvError(null)
    try {
      const pins = Object.values(cvSelectedVersions)
        .filter(Boolean)
        .map((versionId) => ({ entity_type_version_id: versionId }))
      await api.catalogVersions.create({ version_label: newCvLabel.trim(), description: newCvDesc.trim() || undefined, pins: pins.length > 0 ? pins : undefined })
      setCreateCvOpen(false)
      setNewCvLabel('')
      setNewCvDesc('')
      setCvSelectedVersions({})
      loadCatalogVersions()
    } catch (e) {
      setCreateCvError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  // Promote catalog version
  const handlePromote = async (id: string) => {
    try {
      await api.catalogVersions.promote(id)
      loadCatalogVersions()
    } catch (e) {
      setCvError(e instanceof Error ? e.message : 'Failed to promote')
    }
  }

  // Demote catalog version
  const handleDemote = async (id: string, currentStage: string) => {
    const target = currentStage === 'production' ? 'testing' : 'development'
    try {
      await api.catalogVersions.demote(id, target)
      loadCatalogVersions()
    } catch (e) {
      setCvError(e instanceof Error ? e.message : 'Failed to demote')
    }
  }

  // Delete catalog version
  const handleDeleteCatalogVersion = async () => {
    if (!deleteCvTarget) return
    try {
      await api.catalogVersions.delete(deleteCvTarget.id)
      setDeleteCvTarget(null)
      loadCatalogVersions()
    } catch (e) {
      setCvError(e instanceof Error ? e.message : 'Failed to delete')
      setDeleteCvTarget(null)
    }
  }

  const canDeleteCv = (cv: CatalogVersion) => {
    if (role === 'SuperAdmin') return true
    if (role === 'Admin' && cv.lifecycle_stage !== 'production') return true
    return false
  }

  const canCreate = role === 'Admin' || role === 'SuperAdmin'

  const filteredEntityTypes = etFilter
    ? entityTypes.filter((et) => et.name.toLowerCase().includes(etFilter.toLowerCase()))
    : entityTypes

  const stageColor = (stage: string) => {
    switch (stage) {
      case 'development': return 'blue'
      case 'testing': return 'orange'
      case 'production': return 'green'
      default: return 'grey'
    }
  }

  const handleTabSelect = (_e: React.MouseEvent<HTMLElement>, key: string | number) => {
    if (key === 'entityTypes') navigate('/schema')
    else if (key === 'catalogVersions') navigate('/schema/catalog-versions')
    else if (key === 'catalogs') navigate('/schema/catalogs')
    else if (key === 'types') navigate('/schema/types')
    else if (key === 'modelDiagram') navigate('/schema/model-diagram')
  }

  // Entity types list content
  const entityTypesContent = (
    <PageSection padding={{ default: 'noPadding' }}>
      <Title headingLevel="h2" style={{ marginTop: '1rem' }}>Entity Types</Title>

      {etError && <Alert variant="danger" title={etError} isInline style={{ marginBottom: '1rem' }} />}

      <Toolbar>
        <ToolbarContent>
          <ToolbarItem>
            <SearchInput
              placeholder="Filter by name"
              value={etFilter}
              onChange={(_e, value) => setEtFilter(value)}
              onClear={() => setEtFilter('')}
            />
          </ToolbarItem>
          {canCreate && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => setCreateEtOpen(true)}>Create Entity Type</Button>
            </ToolbarItem>
          )}
          <ToolbarItem>
            <Button variant="plain" onClick={loadEntityTypes}>Refresh</Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {etLoading ? (
        <Spinner aria-label="Loading" />
      ) : filteredEntityTypes.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>
            {entityTypes.length === 0
              ? 'No entity types yet. Create one to get started.'
              : 'No entity types match the filter.'}
          </EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Entity types">
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>Description</Th>
              <Th>ID</Th>
              <Th>Created</Th>
              {canCreate && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {filteredEntityTypes.map((et) => (
              <Tr key={et.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/schema/entity-types/${et.id}`)}>
                    {et.name}
                  </Button>
                </Td>
                <Td style={{ maxWidth: '20rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{et.description || '-'}</Td>
                <Td><code>{et.id.slice(0, 8)}...</code></Td>
                <Td>{new Date(et.created_at).toLocaleString()}</Td>
                {canCreate && (
                  <Td>
                    <Button variant="danger" size="sm" onClick={() => setDeleteTarget(et)}>Delete</Button>
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
      <p style={{ marginTop: '0.5rem' }}>Total: {etTotal}</p>
    </PageSection>
  )

  // Catalog versions content
  const catalogVersionsContent = (
    <PageSection padding={{ default: 'noPadding' }}>
      <Title headingLevel="h2" style={{ marginTop: '1rem' }}>Catalog Versions</Title>

      {cvError && <Alert variant="danger" title={cvError} isInline style={{ marginBottom: '1rem' }} />}

      <Toolbar>
        <ToolbarContent>
          <ToolbarItem>
            <Select
              isOpen={stageFilterOpen}
              selected={stageFilter || 'all'}
              onSelect={(_e, value) => { setStageFilter(value === 'all' ? '' : value as string); setStageFilterOpen(false) }}
              onOpenChange={setStageFilterOpen}
              toggle={(ref: React.Ref<MenuToggleElement>) => (
                <MenuToggle ref={ref} onClick={() => setStageFilterOpen(!stageFilterOpen)} isExpanded={stageFilterOpen}>
                  Stage: {stageFilter || 'All'}
                </MenuToggle>
              )}
            >
              <SelectOption value="all">All</SelectOption>
              <SelectOption value="development">Development</SelectOption>
              <SelectOption value="testing">Testing</SelectOption>
              <SelectOption value="production">Production</SelectOption>
            </Select>
          </ToolbarItem>
          {canCreate && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => { setCreateCvOpen(true); loadCvTree() }}>Create Catalog Version</Button>
            </ToolbarItem>
          )}
          <ToolbarItem>
            <Button variant="plain" onClick={loadCatalogVersions}>Refresh</Button>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {cvLoading ? (
        <Spinner aria-label="Loading" />
      ) : catalogVersions.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>No catalog versions yet. Create one to get started.</EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Catalog versions">
          <Thead>
            <Tr>
              <Th>Label</Th>
              <Th>Description</Th>
              <Th>Stage</Th>
              <Th>Created</Th>
              {canCreate && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {catalogVersions.map((cv) => (
              <Tr key={cv.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/schema/catalog-versions/${cv.id}`)}>
                    {cv.version_label}
                  </Button>
                </Td>
                <Td style={{ maxWidth: '20rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{cv.description || '-'}</Td>
                <Td><Label color={stageColor(cv.lifecycle_stage)}>{cv.lifecycle_stage}</Label></Td>
                <Td>{new Date(cv.created_at).toLocaleString()}</Td>
                {canCreate && (
                  <Td>
                    <div style={{ display: 'flex', gap: '0.75rem', minWidth: '20rem' }}>
                      <span style={{ minWidth: '6rem' }}>
                        {cv.lifecycle_stage !== 'production' && (
                          <Button variant="secondary" size="sm" onClick={() => handlePromote(cv.id)}>Promote</Button>
                        )}
                      </span>
                      <span style={{ minWidth: '5.5rem' }}>
                        {cv.lifecycle_stage === 'testing' && (
                          <Button variant="warning" size="sm" onClick={() => handleDemote(cv.id, cv.lifecycle_stage)}>Demote</Button>
                        )}
                        {cv.lifecycle_stage === 'production' && role === 'SuperAdmin' && (
                          <Button variant="warning" size="sm" onClick={() => handleDemote(cv.id, cv.lifecycle_stage)}>Demote</Button>
                        )}
                      </span>
                      {canDeleteCv(cv) && (
                        <Button variant="danger" size="sm" onClick={() => setDeleteCvTarget(cv)}>Delete</Button>
                      )}
                    </div>
                  </Td>
                )}
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
      <p style={{ marginTop: '0.5rem' }}>Total: {cvTotal}</p>
    </PageSection>
  )

  return (
    <Page
      masthead={
        <Masthead>
          <MastheadMain>
            <MastheadBrand onClick={() => navigate('/')} style={{ cursor: 'pointer' }}>{getMastheadTitle()}</MastheadBrand>
          </MastheadMain>
          <MastheadContent>
            <Toolbar>
              <ToolbarContent>
                <ToolbarItem>
                  <Select
                    isOpen={roleSelectOpen}
                    selected={role}
                    onSelect={(_e, value) => {
                      setRole(value as Role)
                      setRoleSelectOpen(false)
                    }}
                    onOpenChange={setRoleSelectOpen}
                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                      <MenuToggle ref={toggleRef} onClick={() => setRoleSelectOpen(!roleSelectOpen)} isExpanded={roleSelectOpen}>
                        Role: {role}
                      </MenuToggle>
                    )}
                  >
                    {ROLES.map((r) => (
                      <SelectOption key={r} value={r}>{r}</SelectOption>
                    ))}
                  </Select>
                </ToolbarItem>
              </ToolbarContent>
            </Toolbar>
          </MastheadContent>
        </Masthead>
      }
    >
      <Routes>
        {/* Landing page */}
        <Route path="/" element={<LandingPage role={role} />} />

        {/* Schema management routes */}
        <Route path="/schema/entity-types/:id" element={<EntityTypeDetailPage role={role} />} />
        <Route path="/schema/catalogs/:name" element={<CatalogDetailPage role={role} />} />
        <Route path="/schema/catalog-versions/:id" element={<CatalogVersionDetailPage role={role} />} />
        <Route path="/schema/types/:id" element={<TypeDefinitionDetailPage role={role} />} />
        <Route path="/schema/*" element={
          <PageSection>
            <Tabs activeKey={activeTab} onSelect={handleTabSelect}>
              <Tab eventKey="entityTypes" title={<TabTitleText>Entity Types</TabTitleText>}>
                {entityTypesContent}
              </Tab>
              <Tab eventKey="catalogVersions" title={<TabTitleText>Catalog Versions</TabTitleText>}>
                {catalogVersionsContent}
              </Tab>
              <Tab eventKey="catalogs" title={<TabTitleText>Catalogs</TabTitleText>}>
                <CatalogListPage role={role} />
              </Tab>
              <Tab eventKey="types" title={<TabTitleText>Types</TabTitleText>}>
                <TypeDefinitionListPage role={role} />
              </Tab>
              <Tab eventKey="modelDiagram" title={<TabTitleText>Model Diagram</TabTitleText>}>
                <PageSection padding={{ default: 'noPadding' }}>
                  {diagramLoading ? (
                    <Spinner />
                  ) : (
                    <EntityTypeDiagram
                      entityTypes={diagramData}
                      onNodeDoubleClick={(entityTypeId) => navigate(`/schema/entity-types/${entityTypeId}`, { state: { from: location.pathname } })}
                      onEdgeClick={handleDiagramEdgeClick}
                    />
                  )}
                </PageSection>
              </Tab>
            </Tabs>
          </PageSection>
        } />

        {/* Catalog data viewer (operational) */}
        <Route path="/catalogs/:name" element={<OperationalCatalogDetailPage role={role} />} />
        <Route path="/catalogs" element={<Navigate to="/" replace />} />
      </Routes>

      {/* Create Entity Type Modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={createEtOpen}
        onClose={() => { setCreateEtOpen(false); setCreateEtError(null) }}
      >
        <ModalHeader title="Create Entity Type" />
        <ModalBody>
          {createEtError && <Alert variant="danger" title={createEtError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="et-name">
              <TextInput id="et-name" value={newEtName} onChange={(_e, v) => setNewEtName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="et-desc">
              <TextInput id="et-desc" value={newEtDesc} onChange={(_e, v) => setNewEtDesc(v)} />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreateEntityType} isDisabled={!newEtName.trim()}>Create</Button>
          <Button variant="link" onClick={() => { setCreateEtOpen(false); setCreateEtError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
      >
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          Are you sure you want to delete entity type <strong>{deleteTarget?.name}</strong>? This action cannot be undone.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDeleteEntityType}>Delete</Button>
          <Button variant="link" onClick={() => setDeleteTarget(null)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Create Catalog Version Modal */}
      <Modal
        variant={ModalVariant.medium}
        isOpen={createCvOpen}
        onClose={() => { setCreateCvOpen(false); setCreateCvError(null); setCvSelectedVersions({}) }}
      >
        <ModalHeader title="Create Catalog Version" />
        <ModalBody>
          {createCvError && <Alert variant="danger" title={createCvError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Version Label" isRequired fieldId="cv-label">
              <TextInput id="cv-label" value={newCvLabel} onChange={(_e, v) => setNewCvLabel(v)} isRequired placeholder="e.g. v1.0" />
            </FormGroup>
            <FormGroup label="Description" fieldId="cv-desc">
              <TextInput id="cv-desc" value={newCvDesc} onChange={(_e, v) => setNewCvDesc(v)} placeholder="Optional description" />
            </FormGroup>
          </Form>
          <Title headingLevel="h4" style={{ marginTop: '1.5rem', marginBottom: '0.5rem' }}>Entity Types to Include</Title>
          {cvEtLoading ? (
            <Spinner aria-label="Loading entity types" />
          ) : cvTree.length === 0 ? (
            <EmptyState><EmptyStateBody>No entity types available.</EmptyStateBody></EmptyState>
          ) : (
            <div>
              {cvTree.map((node) => (
                <TreeNodeRow
                  key={node.entity_type.id}
                  node={node}
                  depth={0}
                  selectedVersions={cvSelectedVersions}
                  onToggle={handleTreeSelect}
                  onVersionChange={(etId, versionId) => {
                    setCvSelectedVersions(prev => ({ ...prev, [etId]: versionId }))
                  }}
                />
              ))}
            </div>
          )}
          {Object.keys(cvSelectedVersions).length > 0 && (
            <Alert variant="info" title={`${Object.keys(cvSelectedVersions).length} entity type(s) selected`} isInline style={{ marginTop: '0.5rem' }} />
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreateCatalogVersion} isDisabled={!newCvLabel.trim()}>Create</Button>
          <Button variant="link" onClick={() => { setCreateCvOpen(false); setCreateCvError(null); setCvSelectedVersions({}) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Catalog Version Confirmation Modal */}
      <Modal
        variant={ModalVariant.small}
        isOpen={deleteCvTarget !== null}
        onClose={() => setDeleteCvTarget(null)}
      >
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          Are you sure you want to delete catalog version <strong>{deleteCvTarget?.version_label}</strong>?
          {deleteCvTarget?.lifecycle_stage === 'production' && (
            <Alert variant="warning" title="This is a production catalog version." isInline style={{ marginTop: '0.5rem' }} />
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDeleteCatalogVersion}>Delete</Button>
          <Button variant="link" onClick={() => setDeleteCvTarget(null)}>Cancel</Button>
        </ModalFooter>
      </Modal>
      {/* Diagram Edit Association Modal */}
      <EditAssociationModal
        isOpen={diagramEditOpen}
        onClose={() => setDiagramEditOpen(false)}
        onSave={handleDiagramEditSave}
        initialData={diagramEditData}
        showEntityTypeNames
        allowTypeChange
      />

    </Page>
  )
}

export default App
