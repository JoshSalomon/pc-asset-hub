import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useAttributeManagement } from './useAttributeManagement'
import type { Attribute, TypeDefinition } from '../types'

vi.mock('../api/client', () => ({
  api: {
    attributes: { list: vi.fn(), add: vi.fn(), remove: vi.fn(), reorder: vi.fn(), edit: vi.fn(), copyFrom: vi.fn() },
    versions: { list: vi.fn() },
    typeDefinitions: { list: vi.fn() },
    entityTypes: { list: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

const mockAttributes: Attribute[] = [
  { id: 'a1', name: 'hostname', description: 'The host', base_type: 'string', ordinal: 0, required: false },
  { id: 'a2', name: 'cpu_count', description: '', base_type: 'number', ordinal: 1, required: false },
]

const mockTypeDefinitions: TypeDefinition[] = [
  { id: 'td1', name: 'Colors', base_type: 'enum', system: false, latest_version: 1, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

function TestComponent({
  entityTypeId,
  attributes,
  typeDefinitions,
}: {
  entityTypeId: string
  attributes: Attribute[]
  typeDefinitions: TypeDefinition[]
}) {
  const onRefresh = vi.fn()
  const setAttributes = vi.fn()
  const setTypeDefinitions = vi.fn()
  const setError = vi.fn()
  const mgmt = useAttributeManagement({
    entityTypeId,
    attributes,
    typeDefinitions,
    onRefresh,
    setAttributes,
    setTypeDefinitions,
    setError,
  })

  return (
    <div>
      <span data-testid="add-open">{String(mgmt.addAttrOpen)}</span>
      <span data-testid="edit-open">{String(mgmt.editAttrOpen)}</span>
      <span data-testid="copy-open">{String(mgmt.copyAttrsOpen)}</span>
      <span data-testid="edit-attr-name">{mgmt.editAttrOrigName}</span>
      <button onClick={() => mgmt.setAddAttrOpen(true)}>Open Add</button>
      <button onClick={() => mgmt.setAddAttrOpen(false)}>Close Add</button>
      <button onClick={() => mgmt.openEditAttr({ id: 'a1', name: 'hostname', description: 'desc', base_type: 'string', ordinal: 0, required: true })}>Open Edit</button>
      <button onClick={() => mgmt.setEditAttrOpen(false)}>Close Edit</button>
      <button onClick={() => mgmt.setCopyAttrsOpen(true)}>Open Copy</button>
      <button onClick={() => mgmt.handleAddAttribute({ name: 'test', description: 'desc', typeDefinitionVersionId: 'tdv1', required: false })}>Add Attr</button>
      <button onClick={() => mgmt.handleRemoveAttribute('hostname')}>Remove Attr</button>
      <button onClick={() => mgmt.handleReorderAttribute(0, 'down')}>Reorder Down</button>
      <button onClick={() => mgmt.handleEditAttribute({ name: 'hostname2', description: 'updated', typeDefinitionVersionId: 'tdv1', required: true })}>Edit Attr</button>
      <button onClick={() => mgmt.handleLoadSourceAttrs('et-2')}>Load Source</button>
      <button onClick={() => mgmt.handleCopyAttributes()}>Copy Attrs</button>
      <span data-testid="source-attrs-count">{mgmt.sourceAttributes.length}</span>
      <span data-testid="copy-source-id">{mgmt.copyAttrsSourceId}</span>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.10: Add attribute modal toggles
test('T-20.10: useAttributeManagement add modal toggles', async () => {
  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await expect.element(page.getByTestId('add-open')).toHaveTextContent('false')
  await page.getByRole('button', { name: 'Open Add' }).click()
  await expect.element(page.getByTestId('add-open')).toHaveTextContent('true')
  await page.getByRole('button', { name: 'Close Add' }).click()
  await expect.element(page.getByTestId('add-open')).toHaveTextContent('false')
})

// T-20.11: Edit attribute opens with pre-populated data
test('T-20.11: useAttributeManagement edit attr opens', async () => {
  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await page.getByRole('button', { name: 'Open Edit' }).click()
  await expect.element(page.getByTestId('edit-open')).toHaveTextContent('true')
  await expect.element(page.getByTestId('edit-attr-name')).toHaveTextContent('hostname')
})

// T-20.12: handleAddAttribute calls API
test('T-20.12: useAttributeManagement add attribute calls api', async () => {
  ;(api.attributes.add as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await page.getByRole('button', { name: 'Add Attr' }).click()
  expect(api.attributes.add).toHaveBeenCalledWith('et-1', expect.objectContaining({ name: 'test' }))
})

// T-20.13: handleRemoveAttribute calls API
test('T-20.13: useAttributeManagement remove attribute calls api', async () => {
  ;(api.attributes.remove as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await page.getByRole('button', { name: 'Remove Attr' }).click()
  expect(api.attributes.remove).toHaveBeenCalledWith('et-1', 'hostname')
})

// T-20.14: handleReorderAttribute calls API with reordered IDs
test('T-20.14: useAttributeManagement reorder calls api', async () => {
  ;(api.attributes.reorder as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await page.getByRole('button', { name: 'Reorder Down' }).click()
  expect(api.attributes.reorder).toHaveBeenCalledWith('et-1', ['a2', 'a1'])
})

// T-20.15: handleEditAttribute calls API
test('T-20.15: useAttributeManagement edit attribute calls api', async () => {
  ;(api.attributes.edit as Mock).mockResolvedValue({})

  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  // First open edit to set editAttrOrigName
  await page.getByRole('button', { name: 'Open Edit' }).click()
  await page.getByRole('button', { name: 'Edit Attr' }).click()
  expect(api.attributes.edit).toHaveBeenCalledWith('et-1', 'hostname', expect.objectContaining({ name: 'hostname2' }))
})

// T-20.16: handleLoadSourceAttrs loads source attributes
test('T-20.16: useAttributeManagement load source attrs', async () => {
  ;(api.attributes.list as Mock).mockResolvedValue({ items: [{ id: 'sa1', name: 'color', type: 'string' }], total: 1 })
  ;(api.versions.list as Mock).mockResolvedValue({ items: [{ version: 2 }], total: 1 })
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [], total: 0 })

  render(<TestComponent entityTypeId="et-1" attributes={mockAttributes} typeDefinitions={mockTypeDefinitions} />)
  await page.getByRole('button', { name: 'Load Source' }).click()
  await expect.element(page.getByTestId('source-attrs-count')).toHaveTextContent('1')
  await expect.element(page.getByTestId('copy-source-id')).toHaveTextContent('et-2')
})

// T-20.17: handleCopyAttributes calls API
test('T-20.17: useAttributeManagement copy attributes calls api', async () => {
  ;(api.attributes.list as Mock).mockResolvedValue({ items: [{ id: 'sa1', name: 'color', type: 'string' }], total: 1 })
  ;(api.versions.list as Mock).mockResolvedValue({ items: [{ version: 2 }], total: 1 })
  ;(api.typeDefinitions.list as Mock).mockResolvedValue({ items: [], total: 0 })
  ;(api.attributes.copyFrom as Mock).mockResolvedValue({})

  // Need a test component that can set selectedCopyAttrs
  function CopyTestComponent() {
    const onRefresh = vi.fn()
    const setAttributes = vi.fn()
    const setTypeDefinitions = vi.fn()
    const setError = vi.fn()
    const mgmt = useAttributeManagement({
      entityTypeId: 'et-1',
      attributes: mockAttributes,
      typeDefinitions: mockTypeDefinitions,
      onRefresh,
      setAttributes,
      setTypeDefinitions,
      setError,
    })
    return (
      <div>
        <button onClick={async () => {
          await mgmt.handleLoadSourceAttrs('et-2')
          mgmt.setSelectedCopyAttrs(['color'])
        }}>Load + Select</button>
        <button onClick={() => mgmt.handleCopyAttributes()}>Copy</button>
        <span data-testid="selected-count">{mgmt.selectedCopyAttrs.length}</span>
      </div>
    )
  }

  render(<CopyTestComponent />)
  await page.getByRole('button', { name: 'Load + Select' }).click()
  await expect.element(page.getByTestId('selected-count')).toHaveTextContent('1')
  await page.getByRole('button', { name: 'Copy' }).click()
  expect(api.attributes.copyFrom).toHaveBeenCalledWith('et-1', expect.objectContaining({
    source_entity_type_id: 'et-2',
    attribute_names: ['color'],
  }))
})
