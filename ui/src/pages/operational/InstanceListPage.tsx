import { PageSection, Title, EmptyState, EmptyStateBody } from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { useCatalogVersion } from '../../context/CatalogVersionContext'

interface Instance {
  id: string
  name: string
  description: string
  version: number
}

interface InstanceListPageProps {
  instances: Instance[]
  total: number
  entityTypeName?: string
}

export function InstanceListPage({ instances, total, entityTypeName }: InstanceListPageProps) {
  const { selectedCatalogVersion } = useCatalogVersion()

  if (!selectedCatalogVersion) {
    return (
      <PageSection>
        <EmptyState>
          <EmptyStateBody>Select a catalog version to view instances.</EmptyStateBody>
        </EmptyState>
      </PageSection>
    )
  }

  return (
    <PageSection>
      <Title headingLevel="h1">{entityTypeName || 'Instances'}</Title>
      <p>Catalog Version: {selectedCatalogVersion}</p>
      <Table aria-label="Instances table">
        <Thead>
          <Tr>
            <Th>Name</Th>
            <Th>Description</Th>
            <Th>Version</Th>
          </Tr>
        </Thead>
        <Tbody>
          {instances.map((inst) => (
            <Tr key={inst.id}>
              <Td>{inst.name}</Td>
              <Td>{inst.description}</Td>
              <Td>{inst.version}</Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
      <p>Total: {total}</p>
    </PageSection>
  )
}
