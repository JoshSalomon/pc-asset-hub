import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Tabs,
  Tab,
  TabTitleText,
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
  Spinner,
  EmptyState,
  EmptyStateBody,
  DescriptionList,
  DescriptionListGroup,
  DescriptionListTerm,
  DescriptionListDescription,
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
  Label,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { ArrowUpIcon, ArrowDownIcon } from '@patternfly/react-icons'
import { api } from '../../api/client'
import type { EntityType, EntityTypeVersion, Attribute, Association, Enum, Role, VersionDiff } from '../../types'

interface Props {
  role: Role
}

export default function EntityTypeDetailPage({ role }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const [entityType, setEntityType] = useState<EntityType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string | number>('overview')

  // Attributes state
  const [attributes, setAttributes] = useState<Attribute[]>([])
  const [attrsLoading, setAttrsLoading] = useState(false)

  // Associations state
  const [associations, setAssociations] = useState<Association[]>([])
  const [assocsLoading, setAssocsLoading] = useState(false)
  const [entityTypes, setEntityTypes] = useState<EntityType[]>([])

  // Versions state
  const [versions, setVersions] = useState<EntityTypeVersion[]>([])
  const [versionsLoading, setVersionsLoading] = useState(false)

  // Enums for attribute creation
  const [enums, setEnums] = useState<Enum[]>([])

  // Add attribute modal
  const [addAttrOpen, setAddAttrOpen] = useState(false)
  const [attrName, setAttrName] = useState('')
  const [attrDesc, setAttrDesc] = useState('')
  const [attrType, setAttrType] = useState('string')
  const [attrTypeOpen, setAttrTypeOpen] = useState(false)
  const [attrEnumId, setAttrEnumId] = useState('')
  const [attrEnumOpen, setAttrEnumOpen] = useState(false)
  const [addAttrError, setAddAttrError] = useState<string | null>(null)

  // Edit attribute modal
  const [editAttrOpen, setEditAttrOpen] = useState(false)
  const [editAttrOrigName, setEditAttrOrigName] = useState('')
  const [editAttrName, setEditAttrName] = useState('')
  const [editAttrDesc, setEditAttrDesc] = useState('')
  const [editAttrType, setEditAttrType] = useState('string')
  const [editAttrTypeOpen, setEditAttrTypeOpen] = useState(false)
  const [editAttrEnumId, setEditAttrEnumId] = useState('')
  const [editAttrEnumOpen, setEditAttrEnumOpen] = useState(false)
  const [editAttrError, setEditAttrError] = useState<string | null>(null)

  // Edit entity type name/description
  const [editNameOpen, setEditNameOpen] = useState(false)
  const [editNameValue, setEditNameValue] = useState('')
  const [editNameError, setEditNameError] = useState<string | null>(null)
  const [deepCopyWarningOpen, setDeepCopyWarningOpen] = useState(false)
  const [pendingNewName, setPendingNewName] = useState('')

  // Copy attributes modal
  const [copyAttrsOpen, setCopyAttrsOpen] = useState(false)
  const [copyAttrsSourceId, setCopyAttrsSourceId] = useState('')
  const [copyAttrsSourceOpen, setCopyAttrsSourceOpen] = useState(false)
  const [sourceAttributes, setSourceAttributes] = useState<Attribute[]>([])
  const [sourceLatestVersion, setSourceLatestVersion] = useState(1)
  const [selectedCopyAttrs, setSelectedCopyAttrs] = useState<string[]>([])
  const [copyAttrsError, setCopyAttrsError] = useState<string | null>(null)

  // Add association modal
  const [addAssocOpen, setAddAssocOpen] = useState(false)
  const [assocTargetId, setAssocTargetId] = useState('')
  const [assocTargetOpen, setAssocTargetOpen] = useState(false)
  const [assocType, setAssocType] = useState('containment')
  const [assocTypeOpen, setAssocTypeOpen] = useState(false)
  const [assocSourceRole, setAssocSourceRole] = useState('')
  const [assocTargetRole, setAssocTargetRole] = useState('')
  const [addAssocError, setAddAssocError] = useState<string | null>(null)

  // Copy modal
  const [copyOpen, setCopyOpen] = useState(false)
  const [copyName, setCopyName] = useState('')
  const [copyError, setCopyError] = useState<string | null>(null)

  // Delete confirmation
  const [deleteOpen, setDeleteOpen] = useState(false)

  // Version diff
  const [diffV1, setDiffV1] = useState<number | ''>('')
  const [diffV2, setDiffV2] = useState<number | ''>('')
  const [diff, setDiff] = useState<VersionDiff | null>(null)
  const [diffError, setDiffError] = useState<string | null>(null)

  const loadEntityType = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError(null)
    try {
      const et = await api.entityTypes.get(id)
      setEntityType(et)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [id])

  const loadAttributes = useCallback(async () => {
    if (!id) return
    setAttrsLoading(true)
    try {
      const res = await api.attributes.list(id)
      setAttributes(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load attributes')
    } finally {
      setAttrsLoading(false)
    }
  }, [id])

  const loadAssociations = useCallback(async () => {
    if (!id) return
    setAssocsLoading(true)
    try {
      const res = await api.associations.list(id)
      setAssociations(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load associations')
    } finally {
      setAssocsLoading(false)
    }
  }, [id])

  const loadVersions = useCallback(async () => {
    if (!id) return
    setVersionsLoading(true)
    try {
      const res = await api.versions.list(id)
      setVersions(res.items || [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load versions')
    } finally {
      setVersionsLoading(false)
    }
  }, [id])

  useEffect(() => {
    loadEntityType()
  }, [loadEntityType])

  useEffect(() => {
    if (activeTab === 'attributes') {
      loadAttributes()
      api.enums.list().then((r) => setEnums(r.items || [])).catch(() => {})
    }
    if (activeTab === 'associations') {
      loadAssociations()
      api.entityTypes.list().then((r) => setEntityTypes(r.items || [])).catch(() => {})
    }
    if (activeTab === 'versions') loadVersions()
  }, [activeTab, loadAttributes, loadAssociations, loadVersions])

  const handleAddAttribute = async () => {
    if (!id || !attrName.trim() || !attrType) return
    setAddAttrError(null)
    try {
      await api.attributes.add(id, {
        name: attrName.trim(),
        description: attrDesc.trim() || undefined,
        type: attrType,
        enum_id: attrType === 'enum' ? attrEnumId : undefined,
      })
      setAddAttrOpen(false)
      setAttrName('')
      setAttrDesc('')
      setAttrType('string')
      setAttrEnumId('')
      loadAttributes()
      loadEntityType()
    } catch (e) {
      setAddAttrError(e instanceof Error ? e.message : 'Failed to add attribute')
    }
  }

  const handleRemoveAttribute = async (name: string) => {
    if (!id) return
    try {
      await api.attributes.remove(id, name)
      loadAttributes()
      loadEntityType()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove attribute')
    }
  }

  const handleReorderAttribute = async (index: number, direction: 'up' | 'down') => {
    if (!id) return
    const newAttrs = [...attributes]
    const swapIndex = direction === 'up' ? index - 1 : index + 1
    if (swapIndex < 0 || swapIndex >= newAttrs.length) return
    ;[newAttrs[index], newAttrs[swapIndex]] = [newAttrs[swapIndex], newAttrs[index]]
    try {
      await api.attributes.reorder(id, newAttrs.map((a) => a.id))
      setAttributes(newAttrs)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to reorder')
    }
  }

  const handleAddAssociation = async () => {
    if (!id || !assocTargetId || !assocType) return
    setAddAssocError(null)
    try {
      await api.associations.create(id, {
        target_entity_type_id: assocTargetId,
        type: assocType,
        source_role: assocSourceRole || undefined,
        target_role: assocTargetRole || undefined,
      })
      setAddAssocOpen(false)
      setAssocTargetId('')
      setAssocType('containment')
      setAssocSourceRole('')
      setAssocTargetRole('')
      loadAssociations()
      loadEntityType()
    } catch (e) {
      setAddAssocError(e instanceof Error ? e.message : 'Failed to create association')
    }
  }

  const handleDeleteAssociation = async (assocId: string) => {
    if (!id) return
    try {
      await api.associations.delete(id, assocId)
      loadAssociations()
      loadEntityType()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete association')
    }
  }

  const handleCopy = async () => {
    if (!id || !copyName.trim()) return
    setCopyError(null)
    try {
      const latestVersion = versions.length > 0 ? Math.max(...versions.map((v) => v.version)) : 1
      await api.entityTypes.copy(id, { source_version: latestVersion, new_name: copyName.trim() })
      setCopyOpen(false)
      setCopyName('')
      navigate('/')
    } catch (e) {
      setCopyError(e instanceof Error ? e.message : 'Failed to copy')
    }
  }

  const handleDelete = async () => {
    if (!id) return
    try {
      await api.entityTypes.delete(id)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to delete')
      setDeleteOpen(false)
    }
  }

  const handleCompareVersions = async () => {
    if (!id || diffV1 === '' || diffV2 === '') return
    setDiffError(null)
    try {
      const result = await api.versions.diff(id, diffV1, diffV2)
      setDiff(result)
    } catch (e) {
      setDiffError(e instanceof Error ? e.message : 'Failed to compare')
      setDiff(null)
    }
  }

  const openEditAttr = (attr: Attribute) => {
    setEditAttrOrigName(attr.name)
    setEditAttrName(attr.name)
    setEditAttrDesc(attr.description || '')
    setEditAttrType(attr.type)
    setEditAttrEnumId(attr.enum_id || '')
    setEditAttrError(null)
    setEditAttrOpen(true)
    if (enums.length === 0) {
      api.enums.list().then((r) => setEnums(r.items || [])).catch(() => {})
    }
  }

  const handleEditAttribute = async () => {
    if (!id) return
    setEditAttrError(null)
    try {
      const data: Record<string, string | undefined> = {}
      if (editAttrName !== editAttrOrigName) data.name = editAttrName
      if (editAttrDesc !== undefined) data.description = editAttrDesc
      data.type = editAttrType
      if (editAttrType === 'enum') data.enum_id = editAttrEnumId
      await api.attributes.edit(id, editAttrOrigName, data)
      setEditAttrOpen(false)
      loadAttributes()
      loadEntityType()
    } catch (e) {
      setEditAttrError(e instanceof Error ? e.message : 'Failed to edit attribute')
    }
  }

  const handleRename = async (deepCopyAllowed = false) => {
    if (!id || !editNameValue.trim()) return
    setEditNameError(null)
    try {
      const result = await api.entityTypes.rename(id, editNameValue.trim(), deepCopyAllowed)
      setEditNameOpen(false)
      setDeepCopyWarningOpen(false)
      if (result.was_deep_copy) {
        navigate(`/entity-types/${result.entity_type.id}`)
      } else {
        loadEntityType()
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to rename'
      if (msg.includes('DEEP_COPY_REQUIRED') || msg.includes('deep_copy_required')) {
        setEditNameOpen(false)
        setPendingNewName(editNameValue.trim())
        setDeepCopyWarningOpen(true)
      } else {
        setEditNameError(msg)
      }
    }
  }

  const handleLoadSourceAttrs = async (sourceId: string) => {
    setCopyAttrsSourceId(sourceId)
    setSelectedCopyAttrs([])
    try {
      // Load source attributes, source versions, and enums in parallel
      const [attrRes, versRes, enumRes] = await Promise.all([
        api.attributes.list(sourceId),
        api.versions.list(sourceId),
        api.enums.list(),
      ])
      setSourceAttributes(attrRes.items || [])
      setEnums(enumRes.items || [])
      // Track the source's latest version for the copy API call
      const srcVersions = versRes.items || []
      const latest = srcVersions.length > 0 ? Math.max(...srcVersions.map((v) => v.version)) : 1
      setSourceLatestVersion(latest)
    } catch {
      setSourceAttributes([])
    }
  }

  const handleCopyAttributes = async () => {
    if (!id || !copyAttrsSourceId || selectedCopyAttrs.length === 0) return
    setCopyAttrsError(null)
    try {
      await api.attributes.copyFrom(id, {
        source_entity_type_id: copyAttrsSourceId,
        source_version: sourceLatestVersion,
        attribute_names: selectedCopyAttrs,
      })
      setCopyAttrsOpen(false)
      setCopyAttrsSourceId('')
      setSourceAttributes([])
      setSelectedCopyAttrs([])
      loadAttributes()
      loadEntityType()
    } catch (e) {
      setCopyAttrsError(e instanceof Error ? e.message : 'Failed to copy attributes')
    }
  }

  if (loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (error && !entityType) return <PageSection><Alert variant="danger" title={error} /></PageSection>
  if (!entityType) return <PageSection><Alert variant="warning" title="Entity type not found" /></PageSection>

  const targetName = (targetId: string) => entityTypes.find((et) => et.id === targetId)?.name || targetId.slice(0, 8)

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate('/')} style={{ marginBottom: '1rem' }}>
        &larr; Back to Entity Types
      </Button>

      {error && <Alert variant="danger" title={error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">{entityType.name}</Title>

      <Tabs activeKey={activeTab} onSelect={(_e, key) => setActiveTab(key)} style={{ marginTop: '1rem' }}>
        {/* Overview Tab */}
        <Tab eventKey="overview" title={<TabTitleText>Overview</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <DescriptionList>
              <DescriptionListGroup>
                <DescriptionListTerm>Name</DescriptionListTerm>
                <DescriptionListDescription>
                  {entityType.name}
                  {canEdit && (
                    <Button variant="link" size="sm" onClick={() => { setEditNameValue(entityType.name); setEditNameError(null); setEditNameOpen(true) }} style={{ marginLeft: '0.5rem' }} aria-label="Edit name">Rename</Button>
                  )}
                </DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>ID</DescriptionListTerm>
                <DescriptionListDescription><code>{entityType.id}</code></DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Created</DescriptionListTerm>
                <DescriptionListDescription>{new Date(entityType.created_at).toLocaleString()}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Updated</DescriptionListTerm>
                <DescriptionListDescription>{new Date(entityType.updated_at).toLocaleString()}</DescriptionListDescription>
              </DescriptionListGroup>
            </DescriptionList>
            {canEdit && (
              <Toolbar style={{ marginTop: '1rem' }}>
                <ToolbarContent>
                  <ToolbarItem>
                    <Button variant="secondary" onClick={() => setCopyOpen(true)}>Copy</Button>
                  </ToolbarItem>
                  <ToolbarItem>
                    <Button variant="danger" onClick={() => setDeleteOpen(true)}>Delete</Button>
                  </ToolbarItem>
                </ToolbarContent>
              </Toolbar>
            )}
          </PageSection>
        </Tab>

        {/* Attributes Tab */}
        <Tab eventKey="attributes" title={<TabTitleText>Attributes</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {canEdit && (
              <Toolbar>
                <ToolbarContent>
                  <ToolbarItem>
                    <Button variant="primary" onClick={() => setAddAttrOpen(true)}>Add Attribute</Button>
                  </ToolbarItem>
                  <ToolbarItem>
                    <Button variant="secondary" onClick={() => { setCopyAttrsOpen(true); api.entityTypes.list().then((r) => setEntityTypes(r.items || [])).catch(() => {}); if (enums.length === 0) api.enums.list().then((r) => setEnums(r.items || [])).catch(() => {}) }}>Copy from...</Button>
                  </ToolbarItem>
                </ToolbarContent>
              </Toolbar>
            )}
            {attrsLoading ? (
              <Spinner aria-label="Loading" />
            ) : attributes.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No attributes defined yet.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Attributes">
                <Thead>
                  <Tr>
                    <Th>Name</Th>
                    <Th>Type</Th>
                    <Th>Description</Th>
                    <Th>Ordinal</Th>
                    {canEdit && <Th>Actions</Th>}
                  </Tr>
                </Thead>
                <Tbody>
                  {attributes.map((attr, idx) => (
                    <Tr key={attr.id}>
                      <Td>{attr.name}</Td>
                      <Td>
                        <Label color={attr.type === 'enum' ? 'purple' : attr.type === 'number' ? 'blue' : 'grey'}>
                          {attr.type === 'enum' && attr.enum_id ? `enum (${enums.find((en) => en.id === attr.enum_id)?.name || attr.enum_id.slice(0, 8)})` : attr.type}
                        </Label>
                      </Td>
                      <Td>{attr.description || '-'}</Td>
                      <Td>{attr.ordinal}</Td>
                      {canEdit && (
                        <Td>
                          <Button
                            variant="plain"
                            size="sm"
                            onClick={() => handleReorderAttribute(idx, 'up')}
                            isDisabled={idx === 0}
                            aria-label="Move up"
                          >
                            <ArrowUpIcon />
                          </Button>
                          <Button
                            variant="plain"
                            size="sm"
                            onClick={() => handleReorderAttribute(idx, 'down')}
                            isDisabled={idx === attributes.length - 1}
                            aria-label="Move down"
                          >
                            <ArrowDownIcon />
                          </Button>
                          <Button variant="secondary" size="sm" onClick={() => openEditAttr(attr)} style={{ marginRight: '0.25rem' }}>Edit</Button>
                          <Button variant="danger" size="sm" onClick={() => handleRemoveAttribute(attr.name)}>Remove</Button>
                        </Td>
                      )}
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </PageSection>
        </Tab>

        {/* Associations Tab */}
        <Tab eventKey="associations" title={<TabTitleText>Associations</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {canEdit && (
              <Toolbar>
                <ToolbarContent>
                  <ToolbarItem>
                    <Button variant="primary" onClick={() => setAddAssocOpen(true)}>Add Association</Button>
                  </ToolbarItem>
                </ToolbarContent>
              </Toolbar>
            )}
            {assocsLoading ? (
              <Spinner aria-label="Loading" />
            ) : associations.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No associations defined yet.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Associations">
                <Thead>
                  <Tr>
                    <Th style={{ width: '10rem' }}>Relationship</Th>
                    <Th>Entity Type</Th>
                    <Th>Role</Th>
                    {canEdit && <Th>Actions</Th>}
                  </Tr>
                </Thead>
                <Tbody>
                  {associations.map((assoc) => {
                    const isIncoming = assoc.direction === 'incoming'
                    const otherEntityId = isIncoming ? assoc.source_entity_type_id : assoc.target_entity_type_id
                    const otherName = targetName(otherEntityId || '')
                    // Show the other entity's role — answers "what role does the other entity play?"
                    const otherRole = isIncoming ? assoc.source_role : assoc.target_role
                    let relationLabel: string
                    let labelColor: 'green' | 'grey' | 'blue' | 'purple' | undefined
                    if (assoc.type === 'bidirectional') {
                      relationLabel = 'references (mutual)'
                      labelColor = 'purple'
                    } else if (assoc.type === 'containment') {
                      relationLabel = isIncoming ? 'contained by' : 'contains'
                      labelColor = isIncoming ? 'grey' : 'green'
                    } else {
                      relationLabel = isIncoming ? 'referenced by' : 'references'
                      labelColor = isIncoming ? 'grey' : 'blue'
                    }
                    return (
                      <Tr key={assoc.id}>
                        <Td><Label color={labelColor}>{relationLabel}</Label></Td>
                        <Td>{otherName}</Td>
                        <Td>{otherRole || '-'}</Td>
                        {canEdit && (
                          <Td>
                            {!isIncoming && <Button variant="danger" size="sm" onClick={() => handleDeleteAssociation(assoc.id)}>Remove</Button>}
                          </Td>
                        )}
                      </Tr>
                    )
                  })}
                </Tbody>
              </Table>
            )}
          </PageSection>
        </Tab>

        {/* Version History Tab */}
        <Tab eventKey="versions" title={<TabTitleText>Version History</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            {versionsLoading ? (
              <Spinner aria-label="Loading" />
            ) : versions.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No versions found.</EmptyStateBody>
              </EmptyState>
            ) : (
              <>
                <Table aria-label="Versions">
                  <Thead>
                    <Tr>
                      <Th>Version</Th>
                      <Th>Description</Th>
                      <Th>Created</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {versions.map((v) => (
                      <Tr key={v.id}>
                        <Td>V{v.version}</Td>
                        <Td>{v.description || '-'}</Td>
                        <Td>{new Date(v.created_at).toLocaleString()}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>

                <Title headingLevel="h4" style={{ marginTop: '1.5rem' }}>Compare Versions</Title>
                <Toolbar>
                  <ToolbarContent>
                    <ToolbarItem>
                      <TextInput
                        type="number"
                        value={diffV1}
                        onChange={(_e, v) => setDiffV1(v ? Number(v) : '')}
                        placeholder="From version"
                        aria-label="From version"
                        style={{ width: '8rem' }}
                      />
                    </ToolbarItem>
                    <ToolbarItem>
                      <TextInput
                        type="number"
                        value={diffV2}
                        onChange={(_e, v) => setDiffV2(v ? Number(v) : '')}
                        placeholder="To version"
                        aria-label="To version"
                        style={{ width: '8rem' }}
                      />
                    </ToolbarItem>
                    <ToolbarItem>
                      <Button
                        variant="secondary"
                        onClick={handleCompareVersions}
                        isDisabled={diffV1 === '' || diffV2 === ''}
                      >
                        Compare
                      </Button>
                    </ToolbarItem>
                  </ToolbarContent>
                </Toolbar>

                {diffError && <Alert variant="danger" title={diffError} isInline style={{ marginTop: '0.5rem' }} />}

                {diff && (
                  <Table aria-label="Version diff" style={{ marginTop: '1rem' }}>
                    <Thead>
                      <Tr>
                        <Th>Name</Th>
                        <Th>Category</Th>
                        <Th>Change</Th>
                        <Th>Old Value</Th>
                        <Th>New Value</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {diff.changes.length === 0 ? (
                        <Tr>
                          <Td colSpan={5}>No differences found.</Td>
                        </Tr>
                      ) : (
                        diff.changes.map((ch, i) => (
                          <Tr key={i}>
                            <Td>{ch.name}</Td>
                            <Td>{ch.category}</Td>
                            <Td>
                              <Label color={ch.change_type === 'added' ? 'green' : ch.change_type === 'removed' ? 'red' : 'blue'}>
                                {ch.change_type}
                              </Label>
                            </Td>
                            <Td>{ch.old_value || '-'}</Td>
                            <Td>{ch.new_value || '-'}</Td>
                          </Tr>
                        ))
                      )}
                    </Tbody>
                  </Table>
                )}
              </>
            )}
          </PageSection>
        </Tab>
      </Tabs>

      {/* Add Attribute Modal */}
      <Modal variant={ModalVariant.small} isOpen={addAttrOpen} onClose={() => { setAddAttrOpen(false); setAddAttrError(null) }}>
        <ModalHeader title="Add Attribute" />
        <ModalBody>
          {addAttrError && <Alert variant="danger" title={addAttrError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="attr-name">
              <TextInput id="attr-name" value={attrName} onChange={(_e, v) => setAttrName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="attr-desc">
              <TextInput id="attr-desc" value={attrDesc} onChange={(_e, v) => setAttrDesc(v)} />
            </FormGroup>
            <FormGroup label="Type" isRequired fieldId="attr-type">
              <Select
                isOpen={attrTypeOpen}
                selected={attrType}
                onSelect={(_e, value) => { setAttrType(value as string); setAttrTypeOpen(false) }}
                onOpenChange={setAttrTypeOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setAttrTypeOpen(!attrTypeOpen)} isExpanded={attrTypeOpen}>{attrType}</MenuToggle>
                )}
              >
                <SelectOption value="string">string</SelectOption>
                <SelectOption value="number">number</SelectOption>
                <SelectOption value="enum">enum</SelectOption>
              </Select>
            </FormGroup>
            {attrType === 'enum' && (
              <FormGroup label="Enum" isRequired fieldId="attr-enum">
                <Select
                  isOpen={attrEnumOpen}
                  selected={attrEnumId}
                  onSelect={(_e, value) => { setAttrEnumId(value as string); setAttrEnumOpen(false) }}
                  onOpenChange={setAttrEnumOpen}
                  toggle={(ref: React.Ref<MenuToggleElement>) => (
                    <MenuToggle ref={ref} onClick={() => setAttrEnumOpen(!attrEnumOpen)} isExpanded={attrEnumOpen}>
                      {enums.find((en) => en.id === attrEnumId)?.name || 'Select enum'}
                    </MenuToggle>
                  )}
                >
                  {enums.map((en) => (
                    <SelectOption key={en.id} value={en.id}>{en.name}</SelectOption>
                  ))}
                </Select>
              </FormGroup>
            )}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleAddAttribute} isDisabled={!attrName.trim()}>Add</Button>
          <Button variant="link" onClick={() => { setAddAttrOpen(false); setAddAttrError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Add Association Modal */}
      <Modal variant={ModalVariant.small} isOpen={addAssocOpen} onClose={() => { setAddAssocOpen(false); setAddAssocError(null) }}>
        <ModalHeader title="Add Association" />
        <ModalBody>
          {addAssocError && <Alert variant="danger" title={addAssocError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Target Entity Type" isRequired fieldId="assoc-target">
              <Select
                isOpen={assocTargetOpen}
                selected={assocTargetId}
                onSelect={(_e, value) => { setAssocTargetId(value as string); setAssocTargetOpen(false) }}
                onOpenChange={setAssocTargetOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setAssocTargetOpen(!assocTargetOpen)} isExpanded={assocTargetOpen}>
                    {entityTypes.find((et) => et.id === assocTargetId)?.name || 'Select target'}
                  </MenuToggle>
                )}
              >
                {entityTypes.filter((et) => et.id !== id).map((et) => (
                  <SelectOption key={et.id} value={et.id}>{et.name}</SelectOption>
                ))}
              </Select>
            </FormGroup>
            <FormGroup label="Type" isRequired fieldId="assoc-type">
              <Select
                isOpen={assocTypeOpen}
                selected={assocType}
                onSelect={(_e, value) => { setAssocType(value as string); setAssocTypeOpen(false) }}
                onOpenChange={setAssocTypeOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setAssocTypeOpen(!assocTypeOpen)} isExpanded={assocTypeOpen}>{assocType}</MenuToggle>
                )}
              >
                <SelectOption value="containment">containment</SelectOption>
                <SelectOption value="directional">directional</SelectOption>
                <SelectOption value="bidirectional">bidirectional</SelectOption>
              </Select>
            </FormGroup>
            <FormGroup label="Source Role" fieldId="assoc-source-role">
              <TextInput id="assoc-source-role" value={assocSourceRole} onChange={(_e, v) => setAssocSourceRole(v)} />
            </FormGroup>
            <FormGroup label="Target Role" fieldId="assoc-target-role">
              <TextInput id="assoc-target-role" value={assocTargetRole} onChange={(_e, v) => setAssocTargetRole(v)} />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleAddAssociation} isDisabled={!assocTargetId}>Add</Button>
          <Button variant="link" onClick={() => { setAddAssocOpen(false); setAddAssocError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Copy Modal */}
      <Modal variant={ModalVariant.small} isOpen={copyOpen} onClose={() => { setCopyOpen(false); setCopyError(null) }}>
        <ModalHeader title="Copy Entity Type" />
        <ModalBody>
          {copyError && <Alert variant="danger" title={copyError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="New Name" isRequired fieldId="copy-name">
              <TextInput id="copy-name" value={copyName} onChange={(_e, v) => setCopyName(v)} isRequired />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCopy} isDisabled={!copyName.trim()}>Copy</Button>
          <Button variant="link" onClick={() => { setCopyOpen(false); setCopyError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Delete Confirmation */}
      <Modal variant={ModalVariant.small} isOpen={deleteOpen} onClose={() => setDeleteOpen(false)}>
        <ModalHeader title="Confirm Deletion" />
        <ModalBody>
          Are you sure you want to delete entity type <strong>{entityType.name}</strong>? This action cannot be undone.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => setDeleteOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Edit Attribute Modal */}
      <Modal variant={ModalVariant.small} isOpen={editAttrOpen} onClose={() => { setEditAttrOpen(false); setEditAttrError(null) }}>
        <ModalHeader title="Edit Attribute" />
        <ModalBody>
          {editAttrError && <Alert variant="danger" title={editAttrError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Name" isRequired fieldId="edit-attr-name">
              <TextInput id="edit-attr-name" value={editAttrName} onChange={(_e, v) => setEditAttrName(v)} isRequired />
            </FormGroup>
            <FormGroup label="Description" fieldId="edit-attr-desc">
              <TextInput id="edit-attr-desc" value={editAttrDesc} onChange={(_e, v) => setEditAttrDesc(v)} />
            </FormGroup>
            <FormGroup label="Type" isRequired fieldId="edit-attr-type">
              <Select
                isOpen={editAttrTypeOpen}
                selected={editAttrType}
                onSelect={(_e, value) => { setEditAttrType(value as string); setEditAttrTypeOpen(false) }}
                onOpenChange={setEditAttrTypeOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setEditAttrTypeOpen(!editAttrTypeOpen)} isExpanded={editAttrTypeOpen}>{editAttrType}</MenuToggle>
                )}
              >
                <SelectOption value="string">string</SelectOption>
                <SelectOption value="number">number</SelectOption>
                <SelectOption value="enum">enum</SelectOption>
              </Select>
            </FormGroup>
            {editAttrType === 'enum' && (
              <FormGroup label="Enum" isRequired fieldId="edit-attr-enum">
                <Select
                  isOpen={editAttrEnumOpen}
                  selected={editAttrEnumId}
                  onSelect={(_e, value) => { setEditAttrEnumId(value as string); setEditAttrEnumOpen(false) }}
                  onOpenChange={setEditAttrEnumOpen}
                  toggle={(ref: React.Ref<MenuToggleElement>) => (
                    <MenuToggle ref={ref} onClick={() => setEditAttrEnumOpen(!editAttrEnumOpen)} isExpanded={editAttrEnumOpen}>
                      {enums.find((en) => en.id === editAttrEnumId)?.name || 'Select enum'}
                    </MenuToggle>
                  )}
                >
                  {enums.map((en) => (
                    <SelectOption key={en.id} value={en.id}>{en.name}</SelectOption>
                  ))}
                </Select>
              </FormGroup>
            )}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleEditAttribute} isDisabled={!editAttrName.trim()}>Save</Button>
          <Button variant="link" onClick={() => { setEditAttrOpen(false); setEditAttrError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Edit Name Modal */}
      <Modal variant={ModalVariant.small} isOpen={editNameOpen} onClose={() => { setEditNameOpen(false); setEditNameError(null) }}>
        <ModalHeader title="Rename Entity Type" />
        <ModalBody>
          {editNameError && <Alert variant="danger" title={editNameError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="New Name" isRequired fieldId="edit-name">
              <TextInput id="edit-name" value={editNameValue} onChange={(_e, v) => setEditNameValue(v)} isRequired />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={() => handleRename(false)} isDisabled={!editNameValue.trim() || editNameValue.trim() === entityType.name}>Rename</Button>
          <Button variant="link" onClick={() => { setEditNameOpen(false); setEditNameError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Deep Copy Warning Modal */}
      <Modal variant={ModalVariant.small} isOpen={deepCopyWarningOpen} onClose={() => setDeepCopyWarningOpen(false)}>
        <ModalHeader title="Deep Copy Required" />
        <ModalBody>
          <Alert variant="warning" title="This entity type is referenced by catalog versions in testing/production." isInline style={{ marginBottom: '1rem' }} />
          Renaming will create a new entity type with the name &quot;{pendingNewName}&quot;. The original entity type will remain unchanged in existing catalog versions.
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={() => { setEditNameValue(pendingNewName); handleRename(true) }}>Create Copy</Button>
          <Button variant="link" onClick={() => setDeepCopyWarningOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Copy Attributes Modal */}
      <Modal variant={ModalVariant.medium} isOpen={copyAttrsOpen} onClose={() => { setCopyAttrsOpen(false); setCopyAttrsError(null) }}>
        <ModalHeader title="Copy Attributes from Another Type" />
        <ModalBody>
          {copyAttrsError && <Alert variant="danger" title={copyAttrsError} isInline style={{ marginBottom: '1rem' }} />}
          <Form>
            <FormGroup label="Source Entity Type" isRequired fieldId="copy-attrs-source">
              <Select
                isOpen={copyAttrsSourceOpen}
                selected={copyAttrsSourceId}
                onSelect={(_e, value) => { handleLoadSourceAttrs(value as string); setCopyAttrsSourceOpen(false) }}
                onOpenChange={setCopyAttrsSourceOpen}
                toggle={(ref: React.Ref<MenuToggleElement>) => (
                  <MenuToggle ref={ref} onClick={() => setCopyAttrsSourceOpen(!copyAttrsSourceOpen)} isExpanded={copyAttrsSourceOpen}>
                    {entityTypes.find((et) => et.id === copyAttrsSourceId)?.name || 'Select source type'}
                  </MenuToggle>
                )}
              >
                {entityTypes.filter((et) => et.id !== id).map((et) => (
                  <SelectOption key={et.id} value={et.id}>{et.name}</SelectOption>
                ))}
              </Select>
            </FormGroup>
          </Form>
          {sourceAttributes.length > 0 && (
            <Table aria-label="Source attributes" style={{ marginTop: '1rem' }}>
              <Thead>
                <Tr>
                  <Th />
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Description</Th>
                  <Th>Status</Th>
                </Tr>
              </Thead>
              <Tbody>
                {sourceAttributes.map((sa) => {
                  const conflict = attributes.some((a) => a.name === sa.name)
                  return (
                    <Tr key={sa.id}>
                      <Td>
                        <input
                          type="checkbox"
                          disabled={conflict}
                          checked={selectedCopyAttrs.includes(sa.name)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedCopyAttrs([...selectedCopyAttrs, sa.name])
                            } else {
                              setSelectedCopyAttrs(selectedCopyAttrs.filter((n) => n !== sa.name))
                            }
                          }}
                        />
                      </Td>
                      <Td>{sa.name}</Td>
                      <Td><Label color={sa.type === 'enum' ? 'purple' : sa.type === 'number' ? 'blue' : 'grey'}>{sa.type === 'enum' && sa.enum_id ? `enum (${enums.find((en) => en.id === sa.enum_id)?.name || sa.enum_id.slice(0, 8)})` : sa.type}</Label></Td>
                      <Td>{sa.description || '-'}</Td>
                      <Td>{conflict ? <Label color="red">Conflict</Label> : <Label color="green">Available</Label>}</Td>
                    </Tr>
                  )
                })}
              </Tbody>
            </Table>
          )}
        </ModalBody>
        <ModalFooter>
          <Button variant="primary" onClick={handleCopyAttributes} isDisabled={selectedCopyAttrs.length === 0}>Copy Selected</Button>
          <Button variant="link" onClick={() => { setCopyAttrsOpen(false); setCopyAttrsError(null) }}>Cancel</Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  )
}
