import { useState, useEffect, useCallback } from 'react'
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
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, CatalogVersionPin, TreeNodeResponse, EntityInstance, ReferenceDetail, Role } from '../../types'

export default function OperationalCatalogDetailPage({ role }: { role: Role }) {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()

  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [pins, setPins] = useState<CatalogVersionPin[]>([])
  const [activeTab, setActiveTab] = useState<string>('overview')

  // Tree state
  const [tree, setTree] = useState<TreeNodeResponse[]>([])
  const [treeLoading, setTreeLoading] = useState(false)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)

  // Selected instance detail
  const [selectedInstance, setSelectedInstance] = useState<EntityInstance | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [forwardRefs, setForwardRefs] = useState<ReferenceDetail[]>([])
  const [reverseRefs, setReverseRefs] = useState<ReferenceDetail[]>([])
  const [refsLoading, setRefsLoading] = useState(false)

  const loadCatalog = useCallback(async () => {
    if (!name) return
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const cat = await api.catalogs.get(name)
      setCatalog(cat)
      const pinsRes = await api.catalogVersions.listPins(cat.catalog_version_id)
      setPins(pinsRes.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalog')
    } finally {
      setLoading(false)
    }
  }, [name, role])

  useEffect(() => { loadCatalog() }, [loadCatalog])

  const loadTree = useCallback(async () => {
    if (!name) return
    setTreeLoading(true)
    try {
      const t = await api.instances.tree(name)
      setTree(t || [])
    } catch {
      setTree([])
    } finally {
      setTreeLoading(false)
    }
  }, [name])

  useEffect(() => {
    if (activeTab === 'tree') loadTree()
  }, [activeTab, loadTree])

  const selectTreeNode = async (node: TreeNodeResponse) => {
    setSelectedNodeId(node.instance_id)
    setDetailLoading(true)
    try {
      const inst = await api.instances.get(name!, node.entity_type_name, node.instance_id)
      setSelectedInstance(inst)
    } catch {
      setSelectedInstance(null)
    } finally {
      setDetailLoading(false)
    }

    // Load references
    setRefsLoading(true)
    try {
      const [fwd, rev] = await Promise.all([
        api.links.forwardRefs(name!, node.entity_type_name, node.instance_id),
        api.links.reverseRefs(name!, node.entity_type_name, node.instance_id),
      ])
      setForwardRefs(fwd || [])
      setReverseRefs(rev || [])
    } catch {
      setForwardRefs([])
      setReverseRefs([])
    } finally {
      setRefsLoading(false)
    }
  }

  const toggleExpand = (nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev)
      if (next.has(nodeId)) {
        next.delete(nodeId)
      } else {
        next.add(nodeId)
      }
      return next
    })
  }

  const navigateToTreeNode = useCallback((instanceId: string) => {
    const findAndSelect = (nodes: TreeNodeResponse[]): boolean => {
      for (const n of nodes) {
        if (n.instance_id === instanceId) {
          selectTreeNode(n)
          return true
        }
        if (n.children && findAndSelect(n.children)) {
          setExpandedNodes(prev => new Set([...prev, n.instance_id]))
          return true
        }
      }
      return false
    }
    findAndSelect(tree)
  }, [tree])

  const browseType = (typeName: string) => {
    setActiveTab('tree')
    // Auto-expand the entity type group
    setExpandedNodes(prev => new Set([...prev, `__group__${typeName}`]))
  }

  // Recursive tree node renderer for instance nodes
  const renderTreeNode = (node: TreeNodeResponse, depth: number) => {
    const hasChildren = node.children && node.children.length > 0
    const isExpanded = expandedNodes.has(node.instance_id)
    const isSelected = selectedNodeId === node.instance_id

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
          onClick={() => selectTreeNode(node)}
        >
          {hasChildren && (
            <span
              style={{ marginRight: '6px', cursor: 'pointer', userSelect: 'none', width: '16px', display: 'inline-block' }}
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.instance_id) }}
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

  // Group root instances by entity type for the tree view
  const groupedTree = (() => {
    const groups: Record<string, TreeNodeResponse[]> = {}
    for (const node of tree) {
      if (!groups[node.entity_type_name]) groups[node.entity_type_name] = []
      groups[node.entity_type_name].push(node)
    }
    return groups
  })()

  // Render entity type group header with expandable children
  const renderEntityTypeGroup = (typeName: string, nodes: TreeNodeResponse[]) => {
    const groupKey = `__group__${typeName}`
    const isExpanded = expandedNodes.has(groupKey)

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
          onClick={() => toggleExpand(groupKey)}
        >
          <span style={{ marginRight: '6px', userSelect: 'none', width: '16px', display: 'inline-block' }}>
            {isExpanded ? '▾' : '▸'}
          </span>
          {typeName} ({nodes.length})
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
      </Title>
      <p style={{ color: '#6a6e73', marginBottom: '1rem' }}>
        Catalog Version: {catalog.catalog_version_label || catalog.catalog_version_id}
        {catalog.description && ` — ${catalog.description}`}
      </p>

      <Tabs activeKey={activeTab} onSelect={(_e, key) => setActiveTab(String(key))} style={{ marginTop: '1rem' }}>
        <Tab eventKey="overview" title={<TabTitleText>Overview</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <Title headingLevel="h3">Entity Types</Title>
            {pins.length === 0 ? (
              <EmptyState><EmptyStateBody>No entity types pinned in this catalog version.</EmptyStateBody></EmptyState>
            ) : (
              <Table aria-label="Entity types">
                <Thead>
                  <Tr>
                    <Th>Entity Type</Th>
                    <Th>Version</Th>
                    <Th>Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {pins.map(pin => (
                    <Tr key={pin.entity_type_name}>
                      <Td>{pin.entity_type_name}</Td>
                      <Td>V{pin.version}</Td>
                      <Td>
                        <Button variant="secondary" size="sm" onClick={() => browseType(pin.entity_type_name)}>
                          Browse Instances
                        </Button>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </PageSection>
        </Tab>

        <Tab eventKey="tree" title={<TabTitleText>Tree Browser</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <div style={{ display: 'flex', gap: '1rem' }}>
              {/* Tree panel (left) */}
              <div style={{ width: '300px', minWidth: '250px', borderRight: '1px solid #d2d2d2', paddingRight: '1rem' }}>
                <Title headingLevel="h4" style={{ marginBottom: '0.5rem' }}>Containment Tree</Title>
                {treeLoading ? (
                  <Spinner size="md" aria-label="Loading tree" />
                ) : tree.length === 0 ? (
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
                {detailLoading ? (
                  <Spinner aria-label="Loading detail" />
                ) : selectedInstance ? (
                  <div>
                    <Title headingLevel="h3">{selectedInstance.name}</Title>

                    {/* Breadcrumb from parent chain */}
                    {selectedInstance.parent_chain && selectedInstance.parent_chain.length > 0 && (
                      <Breadcrumb style={{ marginBottom: '1rem' }}>
                        <BreadcrumbItem>{catalog.name}</BreadcrumbItem>
                        {selectedInstance.parent_chain.map(entry => (
                          <BreadcrumbItem key={entry.instance_id}>
                            {entry.entity_type_name}: {entry.instance_name}
                          </BreadcrumbItem>
                        ))}
                        <BreadcrumbItem isActive>{selectedInstance.name}</BreadcrumbItem>
                      </Breadcrumb>
                    )}

                    <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>{selectedInstance.description}</p>
                    <p style={{ fontSize: '0.85rem', color: '#6a6e73' }}>
                      Version {selectedInstance.version} · Created {new Date(selectedInstance.created_at).toLocaleString()}
                    </p>

                    {/* Attributes */}
                    {selectedInstance.attributes && selectedInstance.attributes.length > 0 && (
                      <div style={{ marginTop: '1rem' }}>
                        <Title headingLevel="h4">Attributes</Title>
                        <Table aria-label="Attributes" variant="compact">
                          <Thead><Tr><Th>Name</Th><Th>Type</Th><Th>Value</Th></Tr></Thead>
                          <Tbody>
                            {selectedInstance.attributes.map(attr => (
                              <Tr key={attr.name}>
                                <Td>{attr.name}</Td>
                                <Td>{attr.type}</Td>
                                <Td>{attr.value != null ? String(attr.value) : '—'}</Td>
                              </Tr>
                            ))}
                          </Tbody>
                        </Table>
                      </div>
                    )}

                    {/* References */}
                    <div style={{ marginTop: '1rem' }}>
                      <Title headingLevel="h4">References</Title>
                      {refsLoading ? (
                        <Spinner size="md" aria-label="Loading references" />
                      ) : (
                        <>
                          {forwardRefs.length > 0 && (
                            <>
                              <p style={{ fontWeight: 600, marginTop: '0.5rem' }}>Forward References</p>
                              <Table aria-label="Forward references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Target</Th><Th>Entity Type</Th></Tr></Thead>
                                <Tbody>
                                  {forwardRefs.map(ref => (
                                    <Tr key={ref.link_id}>
                                      <Td>{ref.association_name}</Td>
                                      <Td>{ref.association_type}</Td>
                                      <Td>
                                        <Button variant="link" isInline onClick={() => navigateToTreeNode(ref.instance_id)}>
                                          {ref.instance_name}
                                        </Button>
                                      </Td>
                                      <Td>{ref.entity_type_name}</Td>
                                    </Tr>
                                  ))}
                                </Tbody>
                              </Table>
                            </>
                          )}
                          {reverseRefs.length > 0 && (
                            <>
                              <p style={{ fontWeight: 600, marginTop: '0.5rem' }}>Referenced By</p>
                              <Table aria-label="Reverse references" variant="compact">
                                <Thead><Tr><Th>Association</Th><Th>Type</Th><Th>Source</Th><Th>Entity Type</Th></Tr></Thead>
                                <Tbody>
                                  {reverseRefs.map(ref => (
                                    <Tr key={ref.link_id}>
                                      <Td>{ref.association_name}</Td>
                                      <Td>{ref.association_type}</Td>
                                      <Td>
                                        <Button variant="link" isInline onClick={() => navigateToTreeNode(ref.instance_id)}>
                                          {ref.instance_name}
                                        </Button>
                                      </Td>
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
      </Tabs>
    </PageSection>
  )
}
