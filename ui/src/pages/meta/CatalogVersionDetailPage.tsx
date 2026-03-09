import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import {
  PageSection,
  Title,
  Tabs,
  Tab,
  TabTitleText,
  Button,
  Alert,
  Spinner,
  EmptyState,
  EmptyStateBody,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Label,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { CatalogVersion, CatalogVersionPin, LifecycleTransition, VersionSnapshot, Role } from '../../types'
import EntityTypeDiagram, { type DiagramEntityType } from '../../components/EntityTypeDiagram'

interface Props {
  role: Role
}

export default function CatalogVersionDetailPage({ role }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const [cv, setCv] = useState<CatalogVersion | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string | number>(searchParams.get('tab') || 'overview')
  const handleTabSelect = (_e: any, key: string | number) => {
    setActiveTab(key)
    setSearchParams({ tab: String(key) }, { replace: true })
  }

  const [pins, setPins] = useState<CatalogVersionPin[]>([])
  const [pinsLoading, setPinsLoading] = useState(false)

  const [transitions, setTransitions] = useState<LifecycleTransition[]>([])
  const [transitionsLoading, setTransitionsLoading] = useState(false)

  const [snapshotOpen, setSnapshotOpen] = useState(false)
  const [snapshot, setSnapshot] = useState<VersionSnapshot | null>(null)
  const [snapshotLoading, setSnapshotLoading] = useState(false)
  const [snapshotError, setSnapshotError] = useState<string | null>(null)

  // Diagram state
  const [diagramData, setDiagramData] = useState<DiagramEntityType[]>([])
  const [diagramLoading, setDiagramLoading] = useState(false)

  const loadCV = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.catalogVersions.get(id)
      setCv(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [id])

  const loadPins = useCallback(async () => {
    if (!id) return
    setPinsLoading(true)
    try {
      const res = await api.catalogVersions.listPins(id)
      setPins(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load pins')
    } finally {
      setPinsLoading(false)
    }
  }, [id])

  const loadTransitions = useCallback(async () => {
    if (!id) return
    setTransitionsLoading(true)
    try {
      const res = await api.catalogVersions.listTransitions(id)
      setTransitions(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load transitions')
    } finally {
      setTransitionsLoading(false)
    }
  }, [id])

  useEffect(() => {
    setAuthRole(role)
    loadCV()
  }, [loadCV, role])

  useEffect(() => {
    if (activeTab === 'bom' || activeTab === 'diagram') loadPins()
    if (activeTab === 'transitions') loadTransitions()
  }, [activeTab, loadPins, loadTransitions])

  const handleOpenSnapshot = async (pin: CatalogVersionPin) => {
    setSnapshotOpen(true)
    setSnapshotLoading(true)
    setSnapshot(null)
    setSnapshotError(null)
    try {
      const data = await api.versions.snapshot(pin.entity_type_id, pin.version)
      setSnapshot(data)
    } catch (e) {
      setSnapshotError(e instanceof Error ? e.message : 'Failed to load snapshot')
    } finally {
      setSnapshotLoading(false)
    }
  }

  const loadDiagramData = useCallback(async () => {
    if (pins.length === 0) return
    setDiagramLoading(true)
    try {
      setAuthRole(role)
      const snapshots: DiagramEntityType[] = await Promise.all(
        pins.map(async (pin) => {
          const snap: VersionSnapshot = await api.versions.snapshot(pin.entity_type_id, pin.version)
          return {
            entityType: { id: pin.entity_type_id, name: pin.entity_type_name, created_at: '', updated_at: '' },
            version: pin.version,
            attributes: snap.attributes || [],
            associations: snap.associations || [],
          }
        })
      )
      setDiagramData(snapshots)
    } catch {
      // Diagram loading failed — will retry on next tab switch
    } finally {
      setDiagramLoading(false)
    }
  }, [pins, role])

  useEffect(() => {
    if (activeTab === 'diagram') loadDiagramData()
  }, [activeTab, loadDiagramData])

  const handlePromote = async () => {
    if (!id) return
    try {
      await api.catalogVersions.promote(id)
      loadCV()
      loadTransitions()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to promote')
    }
  }

  const handleDemote = async () => {
    if (!id || !cv) return
    const target = cv.lifecycle_stage === 'production' ? 'testing' : 'development'
    try {
      await api.catalogVersions.demote(id, target)
      loadCV()
      loadTransitions()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to demote')
    }
  }

  const stageColor = (stage: string) => {
    switch (stage) {
      case 'development': return 'blue'
      case 'testing': return 'orange'
      case 'production': return 'green'
      default: return 'grey'
    }
  }

  const canPromote = cv && cv.lifecycle_stage !== 'production' && (role === 'RW' || role === 'Admin' || role === 'SuperAdmin')
  const canDemote = cv && cv.lifecycle_stage !== 'development' && (
    (cv.lifecycle_stage === 'testing' && role !== 'RO') ||
    (cv.lifecycle_stage === 'production' && role === 'SuperAdmin')
  )

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !cv) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!cv) return <PageSection><Alert variant="warning" title="Catalog version not found" /></PageSection>

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/catalog-versions')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Catalog Versions
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">
        {cv.version_label} <Label color={stageColor(cv.lifecycle_stage)}>{cv.lifecycle_stage}</Label>
      </Title>

      <Tabs activeKey={activeTab} onSelect={handleTabSelect} style={{ marginTop: '1rem' }}>
        {/* Overview Tab */}
        <Tab eventKey="overview" title={<TabTitleText>Overview</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <DescriptionList>
              <DescriptionListGroup>
                <DescriptionListTerm>Version Label</DescriptionListTerm>
                <DescriptionListDescription>{cv.version_label}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Lifecycle Stage</DescriptionListTerm>
                <DescriptionListDescription>
                  <Label color={stageColor(cv.lifecycle_stage)}>{cv.lifecycle_stage}</Label>
                </DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>ID</DescriptionListTerm>
                <DescriptionListDescription><code>{cv.id}</code></DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Created</DescriptionListTerm>
                <DescriptionListDescription>{new Date(cv.created_at).toLocaleString()}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Updated</DescriptionListTerm>
                <DescriptionListDescription>{new Date(cv.updated_at).toLocaleString()}</DescriptionListDescription>
              </DescriptionListGroup>
            </DescriptionList>
            <Toolbar style={{ marginTop: '1rem' }}>
              <ToolbarContent>
                {canPromote && (
                  <ToolbarItem>
                    <Button variant="primary" onClick={handlePromote}>Promote</Button>
                  </ToolbarItem>
                )}
                {canDemote && (
                  <ToolbarItem>
                    <Button variant="warning" onClick={handleDemote}>Demote</Button>
                  </ToolbarItem>
                )}
              </ToolbarContent>
            </Toolbar>
          </PageSection>
        </Tab>

        {/* Bill of Materials Tab */}
        <Tab eventKey="bom" title={<TabTitleText>Bill of Materials</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {pinsLoading ? (
              <Spinner aria-label="Loading" />
            ) : pins.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No entity types pinned to this catalog version.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Pinned entity types">
                <Thead>
                  <Tr>
                    <Th>Entity Type</Th>
                    <Th>Version</Th>
                    <Th>Entity Type ID</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {pins.map((pin) => (
                    <Tr key={pin.entity_type_version_id}>
                      <Td>
                        <Button variant="link" isInline onClick={() => handleOpenSnapshot(pin)}>
                          {pin.entity_type_name}
                        </Button>
                      </Td>
                      <Td>V{pin.version}</Td>
                      <Td><code>{pin.entity_type_id.slice(0, 8)}...</code></Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </PageSection>
        </Tab>

        {/* Transitions Tab */}
        <Tab eventKey="transitions" title={<TabTitleText>Transitions</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {transitionsLoading ? (
              <Spinner aria-label="Loading" />
            ) : transitions.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No transitions recorded.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Lifecycle transitions">
                <Thead>
                  <Tr>
                    <Th>From</Th>
                    <Th>To</Th>
                    <Th>Performed By</Th>
                    <Th>Date</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {transitions.map((lt) => (
                    <Tr key={lt.id}>
                      <Td>{lt.from_stage || '(initial)'}</Td>
                      <Td><Label color={stageColor(lt.to_stage)}>{lt.to_stage}</Label></Td>
                      <Td>{lt.performed_by}</Td>
                      <Td>{new Date(lt.performed_at).toLocaleString()}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </PageSection>
        </Tab>
        <Tab eventKey="diagram" title={<TabTitleText>Diagram</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {(diagramLoading || (pinsLoading && diagramData.length === 0)) ? (
              <Spinner aria-label="Loading diagram" />
            ) : diagramData.length === 0 ? (
              <EmptyState><EmptyStateBody>No entity types pinned.</EmptyStateBody></EmptyState>
            ) : (
              <EntityTypeDiagram entityTypes={diagramData} />
            )}
          </PageSection>
        </Tab>
      </Tabs>

      {/* Version Snapshot Modal */}
      <Modal
        variant={ModalVariant.large}
        isOpen={snapshotOpen}
        onClose={() => { setSnapshotOpen(false); setSnapshot(null); setSnapshotError(null) }}
      >
        <ModalHeader title={snapshot ? `${snapshot.entity_type.name} — V${snapshot.version.version}` : 'Loading...'} />
        <ModalBody>
          {snapshotError && <Alert variant="danger" title={snapshotError} isInline style={{ marginBottom: '1rem' }} />}
          {snapshotLoading ? (
            <Spinner aria-label="Loading snapshot" />
          ) : snapshot ? (
            <>
              <Title headingLevel="h4" style={{ marginBottom: '0.5rem' }}>Attributes</Title>
              {snapshot.attributes.length === 0 ? (
                <EmptyState><EmptyStateBody>No attributes.</EmptyStateBody></EmptyState>
              ) : (
                <Table aria-label="Attributes">
                  <Thead>
                    <Tr>
                      <Th>Name</Th>
                      <Th>Type</Th>
                      <Th>Description</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {snapshot.attributes.map((attr) => (
                      <Tr key={attr.id}>
                        <Td>{attr.name}{attr.required ? ' *' : ''}</Td>
                        <Td>{attr.type === 'enum' && attr.enum_name ? `${attr.enum_name} (enum)` : attr.type}</Td>
                        <Td>{attr.description}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              )}

              <Title headingLevel="h4" style={{ marginTop: '1.5rem', marginBottom: '0.5rem' }}>Associations</Title>
              {snapshot.associations.length === 0 ? (
                <EmptyState><EmptyStateBody>No associations.</EmptyStateBody></EmptyState>
              ) : (
                <Table aria-label="Associations">
                  <Thead>
                    <Tr>
                      <Th>Relationship</Th>
                      <Th>Entity Type</Th>
                      <Th>Name</Th>
                      <Th>Role</Th>
                      <Th>Cardinality</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {snapshot.associations.map((assoc) => {
                      const isOutgoing = assoc.direction === 'outgoing'
                      let relationship: string
                      let labelColor: 'green' | 'grey' | 'blue' | 'purple'
                      if (assoc.type === 'bidirectional') {
                        relationship = 'references (mutual)'
                        labelColor = 'purple'
                      } else if (assoc.type === 'containment') {
                        relationship = isOutgoing ? 'contains' : 'contained by'
                        labelColor = isOutgoing ? 'green' : 'grey'
                      } else {
                        relationship = isOutgoing ? 'references' : 'referenced by'
                        labelColor = isOutgoing ? 'blue' : 'grey'
                      }
                      const otherName = isOutgoing
                        ? (assoc.target_entity_type_name || assoc.target_entity_type_id.slice(0, 8) + '...')
                        : (assoc.source_entity_type_name || assoc.source_entity_type_id?.slice(0, 8) + '...')
                      const role = isOutgoing ? assoc.target_role : assoc.source_role
                      return (
                        <Tr key={assoc.id}>
                          <Td><Label color={labelColor}>{relationship}</Label></Td>
                          <Td>{otherName}</Td>
                          <Td>{assoc.name}</Td>
                          <Td>{role}</Td>
                          <Td>{isOutgoing ? `${assoc.source_cardinality} → ${assoc.target_cardinality}` : `${assoc.target_cardinality} → ${assoc.source_cardinality}`}</Td>
                        </Tr>
                      )
                    })}
                  </Tbody>
                </Table>
              )}
            </>
          ) : null}
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={() => { setSnapshotOpen(false); setSnapshot(null); setSnapshotError(null) }}>Close</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
