import { useState } from 'react'
import {
  PageSection,
  Title,
  Toolbar,
  ToolbarItem,
  ToolbarContent,
  SearchInput,
  Button,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { useAuth } from '../../context/AuthContext'
import type { EntityType } from '../../types'

interface EntityTypeListPageProps {
  entityTypes: EntityType[]
  total: number
  onNavigate?: (id: string) => void
  onRefresh?: () => void
}

export function EntityTypeListPage({
  entityTypes,
  total,
  onNavigate,
}: EntityTypeListPageProps) {
  const { role } = useAuth()
  const [filterText, setFilterText] = useState('')

  const filtered = filterText
    ? entityTypes.filter((et) =>
        et.name.toLowerCase().includes(filterText.toLowerCase())
      )
    : entityTypes

  const canCreate = role === 'Admin' || role === 'SuperAdmin'

  return (
    <PageSection>
      <Title headingLevel="h1">Entity Types</Title>
      <Toolbar>
        <ToolbarContent>
          <ToolbarItem>
            <SearchInput
              placeholder="Filter by name"
              value={filterText}
              onChange={(_e, value) => setFilterText(value)}
              onClear={() => setFilterText('')}
              aria-label="Filter entity types by name"
            />
          </ToolbarItem>
          {canCreate && (
            <ToolbarItem>
              <Button variant="primary">Create Entity Type</Button>
            </ToolbarItem>
          )}
          {canCreate && (
            <ToolbarItem>
              <Button variant="secondary">Copy Entity Type</Button>
            </ToolbarItem>
          )}
        </ToolbarContent>
      </Toolbar>
      <Table aria-label="Entity types table">
        <Thead>
          <Tr>
            <Th>Name</Th>
            <Th>Created</Th>
          </Tr>
        </Thead>
        <Tbody>
          {filtered.map((et) => (
            <Tr
              key={et.id}
              onRowClick={() => onNavigate?.(et.id)}
              isClickable
            >
              <Td>{et.name}</Td>
              <Td>{new Date(et.created_at).toLocaleDateString()}</Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
      <p>Total: {total}</p>
    </PageSection>
  )
}
