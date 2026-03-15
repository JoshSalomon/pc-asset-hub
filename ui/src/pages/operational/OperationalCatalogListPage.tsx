import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  PageSection,
  Title,
  Toolbar,
  ToolbarItem,
  ToolbarContent,
  Button,
  SearchInput,
  Label,
  EmptyState,
  EmptyStateBody,
  Spinner,
  Pagination,
} from '@patternfly/react-core'
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table'
import { api, setAuthRole } from '../../api/client'
import type { Catalog, Role } from '../../types'
import { statusColor } from '../../utils/statusColor'

export default function OperationalCatalogListPage({ role }: { role: Role }) {
  const navigate = useNavigate()
  const [catalogs, setCatalogs] = useState<Catalog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState('')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)

  const loadCatalogs = useCallback(async () => {
    setAuthRole(role)
    setLoading(true)
    setError(null)
    try {
      const res = await api.catalogs.list()
      setCatalogs(res.items || [])
      setTotal(res.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load catalogs')
    } finally {
      setLoading(false)
    }
  }, [role])

  useEffect(() => { loadCatalogs() }, [loadCatalogs])

  const filtered = filter
    ? catalogs.filter(c => c.name.toLowerCase().includes(filter.toLowerCase()))
    : catalogs
  const displayTotal = filtered.length
  const paged = filtered.slice((page - 1) * perPage, page * perPage)

  return (
    <PageSection>
      <Title headingLevel="h1" style={{ marginBottom: '1rem' }}>Catalogs</Title>

      {error && <div style={{ color: '#c9190b', marginBottom: '1rem' }}>{error}</div>}

      <Toolbar>
        <ToolbarContent>
          <ToolbarItem>
            <SearchInput
              placeholder="Filter by name"
              value={filter}
              onChange={(_e, value) => { setFilter(value); setPage(1) }}
              onClear={() => { setFilter(''); setPage(1) }}
            />
          </ToolbarItem>
          <ToolbarItem>
            <Button variant="plain" onClick={loadCatalogs}>Refresh</Button>
          </ToolbarItem>
          <ToolbarItem variant="pagination">
            <Pagination
              itemCount={displayTotal}
              perPage={perPage}
              page={page}
              onSetPage={(_e, p) => setPage(p)}
              onPerPageSelect={(_e, pp) => { setPerPage(pp); setPage(1) }}
              isCompact
            />
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      {loading ? (
        <Spinner aria-label="Loading" />
      ) : paged.length === 0 ? (
        <EmptyState>
          <EmptyStateBody>
            {catalogs.length === 0 ? 'No catalogs available.' : 'No catalogs match the filter.'}
          </EmptyStateBody>
        </EmptyState>
      ) : (
        <Table aria-label="Catalogs">
          <Thead>
            <Tr>
              <Th>Name</Th>
              <Th>Catalog Version</Th>
              <Th>Status</Th>
              <Th>Created</Th>
            </Tr>
          </Thead>
          <Tbody>
            {paged.map((cat) => (
              <Tr key={cat.id}>
                <Td>
                  <Button variant="link" isInline onClick={() => navigate(`/catalogs/${cat.name}`)}>
                    {cat.name}
                  </Button>
                </Td>
                <Td>{cat.catalog_version_label || cat.catalog_version_id.slice(0, 8) + '...'}</Td>
                <Td><Label color={statusColor(cat.validation_status)}>{cat.validation_status}</Label></Td>
                <Td>{new Date(cat.created_at).toLocaleString()}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
      <p style={{ marginTop: '0.5rem' }}>Total: {total}</p>
    </PageSection>
  )
}
