import {
  Title,
  Spinner,
  Breadcrumb,
  BreadcrumbItem,
  Button,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import type { EntityInstance, ReferenceDetail } from '../types'

interface InstanceDetailPanelProps {
  instance: EntityInstance
  catalogName: string
  forwardRefs: ReferenceDetail[]
  reverseRefs: ReferenceDetail[]
  refsLoading: boolean
  onNavigateToRef: (instanceId: string) => void
}

export default function InstanceDetailPanel({
  instance,
  catalogName,
  forwardRefs,
  reverseRefs,
  refsLoading,
  onNavigateToRef,
}: InstanceDetailPanelProps) {
  return (
    <div>
      <Title headingLevel="h3">{instance.name}</Title>

      {/* Breadcrumb from parent chain */}
      {instance.parent_chain && instance.parent_chain.length > 0 && (
        <Breadcrumb style={{ marginBottom: '1rem' }}>
          <BreadcrumbItem>{catalogName}</BreadcrumbItem>
          {instance.parent_chain.map(entry => (
            <BreadcrumbItem key={entry.instance_id}>
              {entry.entity_type_name}: {entry.instance_name}
            </BreadcrumbItem>
          ))}
          <BreadcrumbItem isActive>{instance.name}</BreadcrumbItem>
        </Breadcrumb>
      )}

      <p style={{ color: '#6a6e73', marginBottom: '0.5rem' }}>{instance.description}</p>
      <p style={{ fontSize: '0.85rem', color: '#6a6e73' }}>
        Version {instance.version} · Created {new Date(instance.created_at).toLocaleString()}
      </p>

      {/* Attributes */}
      {instance.attributes && instance.attributes.length > 0 && (
        <div style={{ marginTop: '1rem' }}>
          <Title headingLevel="h4">Attributes</Title>
          <Table aria-label="Attributes" variant="compact">
            <Thead><Tr><Th>Name</Th><Th>Type</Th><Th>Value</Th></Tr></Thead>
            <Tbody>
              {instance.attributes.map(attr => (
                <Tr key={attr.name}>
                  <Td>{attr.name}</Td>
                  <Td>{attr.type}</Td>
                  <Td>{attr.value != null ? String(attr.value) : '\u2014'}</Td>
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
                  <Thead><Tr><Th>Target</Th><Th>Association</Th><Th>Type</Th></Tr></Thead>
                  <Tbody>
                    {forwardRefs.map(ref => (
                      <Tr key={ref.link_id}>
                        <Td>
                          <Button variant="link" isInline onClick={() => onNavigateToRef(ref.instance_id)}>
                            {ref.instance_name} ({ref.entity_type_name})
                          </Button>
                        </Td>
                        <Td>{ref.association_name}</Td>
                        <Td>{ref.association_type}</Td>
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
                  <Thead><Tr><Th>Target</Th><Th>Association</Th><Th>Type</Th></Tr></Thead>
                  <Tbody>
                    {reverseRefs.map(ref => (
                      <Tr key={ref.link_id}>
                        <Td>
                          <Button variant="link" isInline onClick={() => onNavigateToRef(ref.instance_id)}>
                            {ref.instance_name} ({ref.entity_type_name})
                          </Button>
                        </Td>
                        <Td>{ref.association_name}</Td>
                        <Td>{ref.association_type}</Td>
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
  )
}
