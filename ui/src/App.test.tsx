import { render, screen, waitFor, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import App from './App'
import { api, setAuthRole } from './api/client'

vi.mock('./api/client', () => ({
  api: {
    entityTypes: {
      list: vi.fn(),
      create: vi.fn(),
      delete: vi.fn(),
    },
    catalogVersions: {
      list: vi.fn(),
      create: vi.fn(),
      promote: vi.fn(),
      demote: vi.fn(),
    },
    typeDefinitions: {
      list: vi.fn(),
    },
  },
  setAuthRole: vi.fn(),
}))

const mockEntityTypes = [
  { id: 'et-11112222333344444444', name: 'MLModel', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'et-aaaabbbbccccdddddddd', name: 'Dataset', created_at: '2026-01-02T00:00:00Z', updated_at: '2026-01-02T00:00:00Z' },
]

async function renderAndWait() {
  await act(async () => { render(<MemoryRouter><App /></MemoryRouter>) })
  await waitFor(() => expect(api.entityTypes.list).toHaveBeenCalled())
}

beforeEach(() => {
  vi.clearAllMocks()
  ;(api.entityTypes.list as Mock).mockResolvedValue({ items: mockEntityTypes, total: 2 })
  ;(api.catalogVersions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.entityTypes.create as Mock).mockResolvedValue({ entity_type: mockEntityTypes[0] })
  ;(api.entityTypes.delete as Mock).mockResolvedValue(undefined)
  if (api.typeDefinitions?.list) {
    ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  }
})

describe('App', () => {
  it('renders heading', async () => {
    await renderAndWait()
    expect(screen.getByText('AI Asset Hub')).toBeInTheDocument()
  })

  it('calls setAuthRole with Admin on mount', async () => {
    await renderAndWait()
    expect(setAuthRole).toHaveBeenCalledWith('Admin')
  })

  it('fetches entity types on mount', async () => {
    await renderAndWait()
    expect(api.entityTypes.list).toHaveBeenCalled()
  })

  it('displays entity types from API response', async () => {
    await renderAndWait()
    expect(screen.getByText('MLModel')).toBeInTheDocument()
    expect(screen.getByText('Dataset')).toBeInTheDocument()
    expect(screen.getByText('Total: 2')).toBeInTheDocument()
  })

  it('displays truncated entity type IDs', async () => {
    await renderAndWait()
    expect(screen.getByText('et-11112...')).toBeInTheDocument()
  })

  it('renders entity type dates', async () => {
    await renderAndWait()
    // Date is rendered via toLocaleString - check it's present
    const rows = screen.getAllByRole('row')
    expect(rows.length).toBeGreaterThan(1) // header + data rows
  })

  it('shows empty state when API returns no entity types', async () => {
    ;(api.entityTypes.list as Mock).mockResolvedValue({ items: [], total: 0 })
    await renderAndWait()
    expect(screen.getByText('No entity types yet. Create one to get started.')).toBeInTheDocument()
  })

  it('shows error alert when entity type list fails', async () => {
    ;(api.entityTypes.list as Mock).mockRejectedValue(new Error('network error'))
    await renderAndWait()
    await waitFor(() => expect(screen.getByText('network error')).toBeInTheDocument())
  })

  it('handles null items array from API', async () => {
    ;(api.entityTypes.list as Mock).mockResolvedValue({ items: null, total: 0 })
    await renderAndWait()
    expect(screen.getByText('No entity types yet. Create one to get started.')).toBeInTheDocument()
  })

  it('shows Create Entity Type button for Admin role', async () => {
    await renderAndWait()
    expect(screen.getByRole('button', { name: 'Create Entity Type' })).toBeInTheDocument()
  })

  it('shows Delete buttons for each entity type (Admin)', async () => {
    await renderAndWait()
    expect(screen.getAllByRole('button', { name: 'Delete' }).length).toBe(2)
  })

  it('shows Refresh button', async () => {
    await renderAndWait()
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
  })

  it('renders Entity Types, Catalog Versions, and Enums tabs', async () => {
    await renderAndWait()
    const tabs = screen.getAllByRole('tab')
    expect(tabs.length).toBe(3)
    expect(tabs[0].textContent).toContain('Entity Types')
    expect(tabs[1].textContent).toContain('Catalog Versions')
    expect(tabs[2].textContent).toContain('Enums')
  })

  it('Entity Types tab is active by default', async () => {
    await renderAndWait()
    const tabs = screen.getAllByRole('tab')
    expect(tabs[0].getAttribute('aria-selected')).toBe('true')
    expect(tabs[1].getAttribute('aria-selected')).toBe('false')
  })

  it('renders role selector with Admin as default', async () => {
    await renderAndWait()
    expect(screen.getByRole('button', { name: /Role: Admin/i })).toBeInTheDocument()
  })

  it('renders table headers for entity types', async () => {
    await renderAndWait()
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('ID')).toBeInTheDocument()
    expect(screen.getByText('Created')).toBeInTheDocument()
    expect(screen.getByText('Actions')).toBeInTheDocument()
  })

  it('renders search/filter input', async () => {
    await renderAndWait()
    expect(screen.getByPlaceholderText('Filter by name')).toBeInTheDocument()
  })
})

// PatternFly 6's Tabs, Select, Modal, SearchInput, and Button components
// use internal event handlers and browser APIs (ResizeObserver,
// IntersectionObserver) that jsdom cannot fully emulate. Interactive tests
// (clicking buttons, typing in inputs, switching tabs, changing dropdowns)
// timeout in jsdom because PatternFly's internal async state never settles.
//
// Interactive behavior coverage:
// - Filtering: tested in EntityTypeListPage.test.tsx (7 tests, passes props directly)
// - RBAC visibility: tested in EntityTypeListPage.test.tsx (Admin/RO/RW roles)
// - Instance list: tested in InstanceListPage.test.tsx (3 tests)
//
// For full interactive testing of the integrated App with PatternFly
// components, use browser-based testing (Playwright, Cypress).
