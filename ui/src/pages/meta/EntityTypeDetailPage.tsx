import { useState, useCallback } from 'react'
import { useParams, useNavigate, useSearchParams, useLocation } from 'react-router-dom'
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
  Label,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { ArrowUpIcon, ArrowDownIcon } from '@patternfly/react-icons'
import { api } from '../../api/client'
import type { Role, VersionDiff } from '../../types'
import EditAssociationModal from '../../components/EditAssociationModal'
import AddAttributeModal from '../../components/AddAttributeModal'
import EditAttributeModal from '../../components/EditAttributeModal'
import AddAssociationModal from '../../components/AddAssociationModal'
import CopyAttributesModal from '../../components/CopyAttributesModal'
import RenameEntityTypeModal from '../../components/RenameEntityTypeModal'
import { useEntityTypeData } from '../../hooks/useEntityTypeData'
import { useAttributeManagement } from '../../hooks/useAttributeManagement'
import { useAssociationManagement } from '../../hooks/useAssociationManagement'

interface Props {
  role: Role
}

export default function EntityTypeDetailPage({ role }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const [searchParams] = useSearchParams()
  const canEdit = role === 'Admin' || role === 'SuperAdmin'

  const data = useEntityTypeData(id, searchParams.get('tab') || undefined)

  const onRefresh = useCallback(() => {
    data.loadAttributes()
    data.loadEntityType()
  }, [data.loadAttributes, data.loadEntityType])

  const onAssocRefresh = useCallback(() => {
    data.loadAssociations()
    data.loadEntityType()
  }, [data.loadAssociations, data.loadEntityType])

  const attrMgmt = useAttributeManagement({
    entityTypeId: id,
    attributes: data.attributes,
    enums: data.enums,
    onRefresh,
    setAttributes: data.setAttributes,
    setEnums: data.setEnums,
    setError: data.setError,
  })

  const assocMgmt = useAssociationManagement({
    entityTypeId: id,
    onRefresh: onAssocRefresh,
    setError: data.setError,
  })

  // Edit entity type name/description
  const [editNameOpen, setEditNameOpen] = useState(false)
  const [editNameError, setEditNameError] = useState<string | null>(null)
  const [deepCopyWarningOpen, setDeepCopyWarningOpen] = useState(false)
  const [pendingNewName, setPendingNewName] = useState('')

  // Description editing
  const [editingDesc, setEditingDesc] = useState(false)
  const [editDescValue, setEditDescValue] = useState('')

  const handleSaveDescription = async () => {
    if (!id) return
    try {
      await api.entityTypes.update(id, { description: editDescValue })
      setEditingDesc(false)
      data.loadEntityType()
    } catch (e) {
      data.setError(e instanceof Error ? e.message : 'Failed to update description')
    }
  }

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

  const handleRename = async (newName: string, deepCopyAllowed: boolean) => {
    if (!id || !newName.trim()) return
    setEditNameError(null)
    try {
      const result = await api.entityTypes.rename(id, newName.trim(), deepCopyAllowed)
      setEditNameOpen(false)
      setDeepCopyWarningOpen(false)
      if (result.was_deep_copy) {
        navigate(`/schema/entity-types/${result.entity_type.id}`)
      } else {
        data.loadEntityType()
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to rename'
      if (msg.includes('DEEP_COPY_REQUIRED') || msg.includes('deep_copy_required')) {
        setEditNameOpen(false)
        setPendingNewName(newName.trim())
        setDeepCopyWarningOpen(true)
      } else {
        setEditNameError(msg)
      }
    }
  }

  const handleCopy = async () => {
    if (!id || !copyName.trim()) return
    setCopyError(null)
    try {
      const latestVersion = data.versions.length > 0 ? Math.max(...data.versions.map((v) => v.version)) : 1
      await api.entityTypes.copy(id, { source_version: latestVersion, new_name: copyName.trim() })
      setCopyOpen(false)
      setCopyName('')
      navigate('/schema')
    } catch (e) {
      setCopyError(e instanceof Error ? e.message : 'Failed to copy')
    }
  }

  const handleDelete = async () => {
    if (!id) return
    try {
      await api.entityTypes.delete(id)
      navigate('/schema')
    } catch (e) {
      data.setError(e instanceof Error ? e.message : 'Failed to delete')
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

  if (data.loading) return <PageSection><Spinner aria-label="Loading" /></PageSection>
  if (data.error && !data.entityType) return <PageSection><Alert variant="danger" title={data.error} /></PageSection>
  if (!data.entityType) return <PageSection><Alert variant="warning" title="Entity type not found" /></PageSection>

  const targetName = (targetId: string) => data.entityTypes.find((et) => et.id === targetId)?.name || targetId.slice(0, 8)

  return (
    <PageSection>
      <Button variant="link" onClick={() => navigate((location.state as { from?: string })?.from || '/schema')} style={{ marginBottom: '1rem' }}>
        &larr; Back
      </Button>

      {data.error && <Alert variant="danger" title={data.error} isInline style={{ marginBottom: '1rem' }} />}

      <Title headingLevel="h2">{data.entityType.name}</Title>

      <Tabs activeKey={data.activeTab} onSelect={(_e, key) => data.setActiveTab(key)} style={{ marginTop: '1rem' }}>
        {/* Overview Tab */}
        <Tab eventKey="overview" title={<TabTitleText>Overview</TabTitleText>}>
          <PageSection padding={{ default: 'noPadding' }} style={{ marginTop: '1rem' }}>
            <DescriptionList>
              <DescriptionListGroup>
                <DescriptionListTerm>Name</DescriptionListTerm>
                <DescriptionListDescription>
                  {data.entityType.name}
                  {canEdit && (
                    <Button variant="link" size="sm" onClick={() => { setEditNameError(null); setEditNameOpen(true) }} style={{ marginLeft: '0.5rem' }} aria-label="Edit name">Rename</Button>
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
                      {data.entityType.description || <span style={{ color: '#6a6e73' }}>No description</span>}
                      {canEdit && (
                        <Button variant="link" size="sm" onClick={() => { setEditDescValue(data.entityType?.description || ''); setEditingDesc(true) }} style={{ marginLeft: '0.5rem' }} aria-label="Edit description">Edit</Button>
                      )}
                    </>
                  )}
                </DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>ID</DescriptionListTerm>
                <DescriptionListDescription><code>{data.entityType.id}</code></DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Created</DescriptionListTerm>
                <DescriptionListDescription>{new Date(data.entityType.created_at).toLocaleString()}</DescriptionListDescription>
              </DescriptionListGroup>
              <DescriptionListGroup>
                <DescriptionListTerm>Updated</DescriptionListTerm>
                <DescriptionListDescription>{new Date(data.entityType.updated_at).toLocaleString()}</DescriptionListDescription>
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
                    <Button variant="primary" onClick={() => attrMgmt.setAddAttrOpen(true)}>Add Attribute</Button>
                  </ToolbarItem>
                  <ToolbarItem>
                    <Button variant="secondary" onClick={() => { attrMgmt.setCopyAttrsOpen(true); api.entityTypes.list().then((r) => data.setEntityTypes(r.items || [])).catch(() => {}); if (data.enums.length === 0) api.enums.list().then((r) => data.setEnums(r.items || [])).catch(() => {}) }}>Copy from...</Button>
                  </ToolbarItem>
                </ToolbarContent>
              </Toolbar>
            )}
            {data.attrsLoading ? (
              <Spinner aria-label="Loading" />
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
                  {data.attributes.map((attr, idx) => (
                    <Tr key={attr.id || attr.name}>
                      <Td>{attr.name}{attr.required ? ' *' : ''}{attr.system ? <>{' '}<Label color="blue" isCompact>System</Label></> : ''}</Td>
                      <Td>
                        <Label color={attr.type === 'enum' ? 'purple' : attr.type === 'number' ? 'blue' : 'grey'}>
                          {attr.type === 'enum' && attr.enum_id ? `enum (${data.enums.find((en) => en.id === attr.enum_id)?.name || attr.enum_id.slice(0, 8)})` : attr.type}
                        </Label>
                      </Td>
                      <Td>{attr.description || '-'}</Td>
                      <Td>{attr.ordinal}</Td>
                      {canEdit && (
                        <Td>
                          {!attr.system && (
                            <>
                              <Button
                                variant="plain"
                                size="sm"
                                onClick={() => attrMgmt.handleReorderAttribute(idx, 'up')}
                                isDisabled={idx === 0 || data.attributes[idx - 1]?.system}
                                aria-label="Move up"
                              >
                                <ArrowUpIcon />
                              </Button>
                              <Button
                                variant="plain"
                                size="sm"
                                onClick={() => attrMgmt.handleReorderAttribute(idx, 'down')}
                                isDisabled={idx === data.attributes.length - 1 || data.attributes[idx + 1]?.system}
                                aria-label="Move down"
                              >
                                <ArrowDownIcon />
                              </Button>
                              <Button variant="secondary" size="sm" onClick={() => attrMgmt.openEditAttr(attr)} style={{ marginRight: '0.25rem' }}>Edit</Button>
                              <Button variant="danger" size="sm" onClick={() => attrMgmt.handleRemoveAttribute(attr.name)}>Remove</Button>
                            </>
                          )}
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
                    <Button variant="primary" onClick={() => assocMgmt.setAddAssocOpen(true)}>Add Association</Button>
                  </ToolbarItem>
                </ToolbarContent>
              </Toolbar>
            )}
            {data.assocsLoading ? (
              <Spinner aria-label="Loading" />
            ) : data.associations.length === 0 ? (
              <EmptyState>
                <EmptyStateBody>No associations defined yet.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Associations">
                <Thead>
                  <Tr>
                    <Th>Relationship</Th>
                    <Th>Entity Type</Th>
                    <Th>Name</Th>
                    <Th>Role</Th>
                    <Th>Cardinality</Th>
                    {canEdit && <Th>Actions</Th>}
                  </Tr>
                </Thead>
                <Tbody>
                  {data.associations.map((assoc) => {
                    const isIncoming = assoc.direction === 'incoming'
                    const otherEntityId = isIncoming ? assoc.source_entity_type_id : assoc.target_entity_type_id
                    const otherName = targetName(otherEntityId || '')
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
                        <Td>{assoc.name}</Td>
                        <Td>{otherRole || '-'}</Td>
                        <Td>{isIncoming ? `${assoc.target_cardinality} → ${assoc.source_cardinality}` : `${assoc.source_cardinality} → ${assoc.target_cardinality}`}</Td>
                        {canEdit && (
                          <Td>
                            {!isIncoming && (
                              <>
                                <Button variant="secondary" size="sm" onClick={() => assocMgmt.openEditAssoc(assoc)} style={{ marginRight: '0.5rem' }}>Edit</Button>
                                <Button variant="danger" size="sm" onClick={() => assocMgmt.handleDeleteAssociation(assoc.name)}>Remove</Button>
                              </>
                            )}
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
            {data.versionsLoading ? (
              <Spinner aria-label="Loading" />
            ) : data.versions.length === 0 ? (
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
                    {data.versions.map((v) => (
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
      <AddAttributeModal
        isOpen={attrMgmt.addAttrOpen}
        onClose={() => { attrMgmt.setAddAttrOpen(false); attrMgmt.setAddAttrError(null) }}
        onSubmit={attrMgmt.handleAddAttribute}
        enums={data.enums}
        error={attrMgmt.addAttrError}
      />

      {/* Add Association Modal */}
      <AddAssociationModal
        isOpen={assocMgmt.addAssocOpen}
        onClose={() => { assocMgmt.setAddAssocOpen(false); assocMgmt.setAddAssocError(null) }}
        onSubmit={assocMgmt.handleAddAssociation}
        entityTypes={data.entityTypes}
        currentEntityTypeId={id}
        error={assocMgmt.addAssocError}
      />

      {/* Edit Association Modal */}
      <EditAssociationModal
        isOpen={assocMgmt.editAssocOpen}
        onClose={() => assocMgmt.setEditAssocOpen(false)}
        onSave={assocMgmt.handleEditAssociationSave}
        initialData={assocMgmt.editAssocData}
        allowTypeChange
      />

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
          Are you sure you want to delete entity type <strong>{data.entityType.name}</strong>? This action cannot be undone.
        </ModalBody>
        <ModalFooter>
          <Button variant="danger" onClick={handleDelete}>Delete</Button>
          <Button variant="link" onClick={() => setDeleteOpen(false)}>Cancel</Button>
        </ModalFooter>
      </Modal>

      {/* Edit Attribute Modal */}
      {(() => {
        const editingAttr = data.attributes.find(a => a.name === attrMgmt.editAttrOrigName)
        return (
          <EditAttributeModal
            isOpen={attrMgmt.editAttrOpen}
            onClose={() => { attrMgmt.setEditAttrOpen(false); attrMgmt.setEditAttrError(null) }}
            onSubmit={attrMgmt.handleEditAttribute}
            enums={data.enums}
            error={attrMgmt.editAttrError}
            initialName={attrMgmt.editAttrOrigName}
            initialDescription={editingAttr?.description || ''}
            initialType={editingAttr?.type || 'string'}
            initialEnumId={editingAttr?.enum_id || ''}
            initialRequired={editingAttr?.required || false}
          />
        )
      })()}

      {/* Rename Entity Type Modal */}
      <RenameEntityTypeModal
        isOpen={editNameOpen}
        onClose={() => { setEditNameOpen(false); setEditNameError(null) }}
        onSubmit={handleRename}
        currentName={data.entityType.name}
        error={editNameError}
        deepCopyWarningOpen={deepCopyWarningOpen}
        pendingNewName={pendingNewName}
        onDeepCopyConfirm={() => handleRename(pendingNewName, true)}
        onDeepCopyCancel={() => setDeepCopyWarningOpen(false)}
      />

      {/* Copy Attributes Modal */}
      <CopyAttributesModal
        isOpen={attrMgmt.copyAttrsOpen}
        onClose={() => { attrMgmt.setCopyAttrsOpen(false); attrMgmt.setCopyAttrsError(null) }}
        onSubmit={attrMgmt.handleCopyAttributes}
        onLoadSource={attrMgmt.handleLoadSourceAttrs}
        entityTypes={data.entityTypes}
        currentEntityTypeId={id}
        sourceAttributes={attrMgmt.sourceAttributes}
        existingAttributes={data.attributes}
        enums={data.enums}
        error={attrMgmt.copyAttrsError}
      />
    </PageSection>
  )
}
