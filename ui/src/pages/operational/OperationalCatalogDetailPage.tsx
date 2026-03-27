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
  Alert,
  EmptyState,
  EmptyStateBody,
  Spinner,
  Breadcrumb,
  BreadcrumbItem,
} from '@patternfly/react-core'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, TreeNodeResponse, Role } from '../../types'
import { useValidation } from '../../hooks/useValidation'
import { useContainmentTree } from '../../hooks/useContainmentTree'
import { useCatalogDiagram } from '../../hooks/useCatalogDiagram'
import ValidationResults from '../../components/ValidationResults'
import InstanceDetailPanel from '../../components/InstanceDetailPanel'
import EntityTypeDiagram from '../../components/EntityTypeDiagram'

export default function OperationalCatalogDetailPage({ role }: { role: Role }) {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()

  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string>('tree')

  const ct = useContainmentTree(name)

  const loadCatalog = useCallback(async () => {
    if (!name) return
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const cat = await api.catalogs.get(name)
      setCatalog(cat)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalog')
    } finally {
      setLoading(false)
    }
  }, [name, role])

  useEffect(() => { loadCatalog() }, [loadCatalog])

  const validation = useValidation(name, loadCatalog)
  const diagram = useCatalogDiagram(catalog?.catalog_version_id)

  useEffect(() => {
    if (activeTab === 'tree') ct.loadTree()
    if (activeTab === '__diagram__') diagram.loadDiagram()
  }, [activeTab, ct.loadTree, diagram.loadDiagram])

  // Recursive tree node renderer for instance nodes
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
              {isExpanded ? '\u25BE' : '\u25B8'}
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
    for (const node of ct.tree) {
      if (!groups[node.entity_type_name]) groups[node.entity_type_name] = []
      groups[node.entity_type_name].push(node)
    }
    return groups
  })()

  // Render entity type group header with expandable children
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
            {isExpanded ? '\u25BE' : '\u25B8'}
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
      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>
        Catalog Version: {catalog.catalog_version_label || catalog.catalog_version_id}
        {catalog.description && ` \u2014 ${catalog.description}`}
      </p>

      {(role === 'RW' || role === 'Admin' || role === 'SuperAdmin') && (
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
                <Title headingLevel="h4" style={{ marginBottom: '0.5rem' }}>Containment Tree</Title>
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
                  <InstanceDetailPanel
                    instance={ct.selectedInstance}
                    catalogName={catalog.name}
                    forwardRefs={ct.forwardRefs}
                    reverseRefs={ct.reverseRefs}
                    refsLoading={ct.refsLoading}
                    onNavigateToRef={ct.navigateToTreeNode}
                  />
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
        </Tab>
      </Tabs>
    </PageSection>
  )
}
