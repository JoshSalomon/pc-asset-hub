import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { EntityTypeListPage } from './EntityTypeListPage'
import { AuthProvider } from '../../context/AuthContext'
import type { EntityType } from '../../types'

const mockEntityTypes: EntityType[] = [
  { id: '1', name: 'Model', created_at: '2024-01-01', updated_at: '2024-01-01' },
  { id: '2', name: 'Tool', created_at: '2024-01-02', updated_at: '2024-01-02' },
  { id: '3', name: 'Prompt', created_at: '2024-01-03', updated_at: '2024-01-03' },
]

function renderWithAuth(ui: React.ReactElement, role: 'RO' | 'RW' | 'Admin' | 'SuperAdmin' = 'Admin') {
  return render(<AuthProvider initialRole={role}>{ui}</AuthProvider>)
}

// T-7.01: List renders all entity types with name, version, description, counts
describe('T-7.01: EntityTypeListPage renders all types', () => {
  it('shows all entity types', () => {
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} />
    )
    expect(screen.getByText('Model')).toBeInTheDocument()
    expect(screen.getByText('Tool')).toBeInTheDocument()
    expect(screen.getByText('Prompt')).toBeInTheDocument()
    expect(screen.getByText('Total: 3')).toBeInTheDocument()
  })
})

// T-7.02: Filter by name
describe('T-7.02: Filter by name', () => {
  it('filters entity types when text is entered', async () => {
    const user = userEvent.setup()
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} />
    )

    const searchInput = screen.getByPlaceholderText('Filter by name')
    await user.type(searchInput, 'Model')

    expect(screen.getByText('Model')).toBeInTheDocument()
    expect(screen.queryByText('Tool')).not.toBeInTheDocument()
  })
})

// T-7.03: Sort by name/version — list reorders (basic sort via data)
describe('T-7.03: Sorting', () => {
  it('renders entity types in provided order', () => {
    const sorted = [...mockEntityTypes].sort((a, b) => b.name.localeCompare(a.name))
    renderWithAuth(
      <EntityTypeListPage entityTypes={sorted} total={3} />
    )
    const rows = screen.getAllByRole('row')
    // First data row (after header) should be Tool (sorted desc)
    expect(rows[1]).toHaveTextContent('Tool')
  })
})

// T-7.04: Click row navigates to detail view
describe('T-7.04: Row click navigation', () => {
  it('calls onNavigate when row is clicked', async () => {
    const user = userEvent.setup()
    const onNavigate = vi.fn()
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} onNavigate={onNavigate} />
    )

    await user.click(screen.getByText('Model'))
    expect(onNavigate).toHaveBeenCalledWith('1')
  })
})

// T-7.05: Create/copy buttons visible for Admin, hidden for RO/RW
describe('T-7.05: Role-aware action buttons', () => {
  it('shows Create and Copy buttons for Admin', () => {
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} />,
      'Admin'
    )
    expect(screen.getByText('Create Entity Type')).toBeInTheDocument()
    expect(screen.getByText('Copy Entity Type')).toBeInTheDocument()
  })

  it('hides Create and Copy buttons for RO', () => {
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} />,
      'RO'
    )
    expect(screen.queryByText('Create Entity Type')).not.toBeInTheDocument()
    expect(screen.queryByText('Copy Entity Type')).not.toBeInTheDocument()
  })

  it('hides Create and Copy buttons for RW', () => {
    renderWithAuth(
      <EntityTypeListPage entityTypes={mockEntityTypes} total={3} />,
      'RW'
    )
    expect(screen.queryByText('Create Entity Type')).not.toBeInTheDocument()
    expect(screen.queryByText('Copy Entity Type')).not.toBeInTheDocument()
  })
})
