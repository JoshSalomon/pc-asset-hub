import '@patternfly/patternfly/patternfly.css'
import { useEffect, useState, useCallback } from 'react'
import { Routes, Route, useNavigate, useLocation } from 'react-router-dom'
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
import type { EntityType, CatalogVersion, Role } from './types'
import EntityTypeDetailPage from './pages/meta/EntityTypeDetailPage'
import EnumListPage from './pages/meta/EnumListPage'
import EnumDetailPage from './pages/meta/EnumDetailPage'

const ROLES: Role[] = ['RO', 'RW', 'Admin', 'SuperAdmin']

function App() {
  const navigate = useNavigate()
  const location = useLocation()
  const [role, setRole] = useState<Role>('Admin')
  const [roleSelectOpen, setRoleSelectOpen] = useState(false)

  // Determine active tab from URL
  const getActiveTab = () => {
    if (location.pathname.startsWith('/catalog-versions')) return 'catalogVersions'
    if (location.pathname.startsWith('/enums')) return 'enums'
    return 'entityTypes'
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

  // Create Catalog Version modal
  const [createCvOpen, setCreateCvOpen] = useState(false)
  const [newCvLabel, setNewCvLabel] = useState('')
  const [createCvError, setCreateCvError] = useState<string | null>(null)

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
      const res = await api.catalogVersions.list()
      setCatalogVersions(res.items || [])
      setCvTotal(res.total)
    } catch (e) {
      setCvError(e instanceof Error ? e.message : 'Failed to load catalog versions')
    } finally {
      setCvLoading(false)
    }
  }, [])

  // Load data on mount and tab change
  const activeTab = getActiveTab()
  useEffect(() => {
    setAuthRole(role)
    if (activeTab === 'entityTypes') loadEntityTypes()
    if (activeTab === 'catalogVersions') loadCatalogVersions()
  }, [activeTab, role, loadEntityTypes, loadCatalogVersions])

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

  // Create catalog version
  const handleCreateCatalogVersion = async () => {
    if (!newCvLabel.trim()) return
    setCreateCvError(null)
    try {
      await api.catalogVersions.create({ version_label: newCvLabel.trim() })
      setCreateCvOpen(false)
      setNewCvLabel('')
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
    if (key === 'entityTypes') navigate('/')
    else if (key === 'catalogVersions') navigate('/catalog-versions')
    else if (key === 'enums') navigate('/enums')
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
              <Th>ID</Th>
              <Th>Created</Th>
              {canCreate && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {filteredEntityTypes.map((et) => (
              <Tr key={et.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/entity-types/${et.id}`)}>
                    {et.name}
                  </Button>
                </Td>
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
          {canCreate && (
            <ToolbarItem>
              <Button variant="primary" onClick={() => setCreateCvOpen(true)}>Create Catalog Version</Button>
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
              <Th>Stage</Th>
              <Th>Created</Th>
              {canCreate && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {catalogVersions.map((cv) => (
              <Tr key={cv.id}>
                <Td>{cv.version_label}</Td>
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
                        {cv.lifecycle_stage !== 'development' && (
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
            <MastheadBrand>AI Asset Hub</MastheadBrand>
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
        <Route path="/entity-types/:id" element={<EntityTypeDetailPage role={role} />} />
        <Route path="/enums/:id" element={<EnumDetailPage role={role} />} />
        <Route path="*" element={
          <PageSection>
            <Tabs activeKey={activeTab} onSelect={handleTabSelect}>
              <Tab eventKey="entityTypes" title={<TabTitleText>Entity Types</TabTitleText>}>
                {entityTypesContent}
              </Tab>
              <Tab eventKey="catalogVersions" title={<TabTitleText>Catalog Versions</TabTitleText>}>
                {catalogVersionsContent}
              </Tab>
              <Tab eventKey="enums" title={<TabTitleText>Enums</TabTitleText>}>
                <EnumListPage role={role} />
              </Tab>
            </Tabs>
          </PageSection>
        } />
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
        variant={ModalVariant.small}
        isOpen={createCvOpen}
        onClose={() => { setCreateCvOpen(false); setCreateCvError(null) }}
      >
        <ModalHeader title="Create Catalog Version" />
        <ModalBody>
          {createCvError && <Alert variant="danger" title={createCvError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Version Label" isRequired fieldId="cv-label">
              <TextInput id="cv-label" value={newCvLabel} onChange={(_e, v) => setNewCvLabel(v)} isRequired placeholder="e.g. v1.0" />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCreateCatalogVersion} isDisabled={!newCvLabel.trim()}>Create</Button>
          <Button variant="link" onClick={() => { setCreateCvOpen(false); setCreateCvError(null) }}>Cancel</Button>
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
    </Page>
  )
}

export default App
