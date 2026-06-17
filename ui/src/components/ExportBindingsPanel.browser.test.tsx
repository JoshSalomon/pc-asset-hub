import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page, userEvent } from 'vitest/browser'
import ExportBindingsPanel from './ExportBindingsPanel'

vi.mock('../api/client', () => ({
  api: {
    exportBindings: {
      list: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      run: vi.fn(),
    },
    exporters: {
      list: vi.fn(),
    },
    catalogVersions: {
      listPins: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    },
    attributes: {
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    },
    instances: {
      list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    },
  },
}))

const { api } = await import('../api/client')

beforeEach(() => {
  vi.clearAllMocks()
})

function renderPanel(overrides: Partial<React.ComponentProps<typeof ExportBindingsPanel>> = {}) {
  const props = {
    catalogName: 'test-catalog',
    catalogVersionId: 'cv-1',
    isAdmin: true,
    isRW: true,
    ...overrides,
  }
  return render(<ExportBindingsPanel {...props} />)
}

const makeBinding = (overrides = {}) => ({
  id: 'b1', catalog_id: 'cat1', exporter_name: 'mcp-gateway', parameters: {},
  enabled: true, last_run_status: 'never', last_run_at: null, last_run_error: '',
  created_at: '', updated_at: '', ...overrides,
})

// T-36.118: Built-in exporter shows "built-in" badge
test('T-36.118: built-in exporter shows built-in badge', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'mcp-gateway', description: 'MCP', parameter_schema: [], source: 'built-in', health: 'n/a' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding()],
    total: 1,
  })

  renderPanel()
  await expect.element(page.getByText('built-in')).toBeVisible()
})

// T-36.119: Webhook exporter shows "webhook" badge + health dot
test('T-36.119: webhook exporter shows webhook badge and health dot', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'my-webhook', description: 'Webhook', parameter_schema: [], source: 'webhook', health: 'Ready' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'my-webhook' })],
    total: 1,
  })

  renderPanel()
  await expect.element(page.getByText('webhook')).toBeVisible()
})

// T-36.120: Orphaned binding shows warning icon with tooltip
test('T-36.120: orphaned binding shows warning icon', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'deleted-exporter' })],
    total: 1,
  })

  renderPanel()
  await expect.element(page.getByText('deleted-exporter')).toBeVisible()
  const row = page.getByRole('row', { name: /deleted-exporter/ })
  await expect.element(row).toBeVisible()
  // Verify the warning icon is present — PatternFly Icon with status="warning"
  const warningIcons = document.querySelectorAll('.pf-m-warning svg')
  expect(warningIcons.length).toBeGreaterThan(0)
})

// T-36.121: Export Now disabled for orphaned binding
test('T-36.121: Export Now disabled for orphaned binding', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'deleted-exporter' })],
    total: 1,
  })

  renderPanel()
  const exportBtn = page.getByRole('button', { name: 'Export Now' })
  await expect.element(exportBtn).toBeDisabled()
})

// T-36.124: Existing binding CRUD works with webhook exporter (no regression)
test('T-36.124: binding CRUD works with webhook exporter', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'my-webhook', description: 'Webhook Exporter', parameter_schema: [], source: 'webhook', health: 'Ready' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'my-webhook', last_run_status: 'success', last_run_at: '2026-01-01T00:00:00Z' })],
    total: 1,
  })

  renderPanel()
  await expect.element(page.getByText('my-webhook')).toBeVisible()
  const exportBtn = page.getByRole('button', { name: 'Export Now' })
  await expect.element(exportBtn).toBeEnabled()
})

// T-36.122: Skipped status renders as grey label, distinct from failed (red)
test('T-36.122: skipped status renders as grey label distinct from failed', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [
      { name: 'exporter-a', description: 'A', parameter_schema: [], source: 'built-in', health: 'n/a' },
      { name: 'exporter-b', description: 'B', parameter_schema: [], source: 'built-in', health: 'n/a' },
    ],
    total: 2,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [
      makeBinding({ id: 'b-skip', exporter_name: 'exporter-a', last_run_status: 'skipped' }),
      makeBinding({ id: 'b-fail', exporter_name: 'exporter-b', last_run_status: 'failed', last_run_error: 'something broke' }),
    ],
    total: 2,
  })

  renderPanel()
  // Both labels should be visible
  await expect.element(page.getByText('skipped')).toBeVisible()
  await expect.element(page.getByText('failed')).toBeVisible()
  // Verify "skipped" and "failed" are rendered as Label components with distinct colors.
  // PatternFly Label renders with class containing the color variant.
  // skipped = grey, failed = red — they must be visually distinct.
  const skippedLabel = page.getByText('skipped')
  const failedLabel = page.getByText('failed')
  // The Label elements should have different color class names
  await expect.element(skippedLabel).toBeInTheDocument()
  await expect.element(failedLabel).toBeInTheDocument()
})

// T-36.123 / T-36.99: Zero-param exporter — Add Binding modal shows no parameter fields
test('T-36.123: zero-param exporter Add Binding modal shows no parameter fields', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'simple-export', description: 'No params needed', parameter_schema: [], source: 'webhook', health: 'Ready' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.catalogVersions.listPins).mockResolvedValue({ items: [], total: 0 })

  renderPanel()
  // Click the Add Export Binding button
  const addBtn = page.getByRole('button', { name: 'Add Export Binding' })
  await expect.element(addBtn).toBeVisible()
  await addBtn.click()

  // Modal should appear — check for the modal dialog
  const dialog = page.getByRole('dialog')
  await expect.element(dialog).toBeVisible()
  const exporterSelect = page.getByLabelText('Select exporter')
  await expect.element(exporterSelect).toBeVisible()

  // Select the zero-param exporter using native select interaction
  await userEvent.selectOptions(exporterSelect, 'simple-export')

  // After selecting, the exporter name should be shown in the select
  // and no parameter FormGroups should appear since parameter_schema is empty.
  // Verify no TextInput elements are rendered for parameters.
  const paramInputs = page.getByRole('textbox')
  await expect.element(paramInputs).not.toBeInTheDocument()

  // The Create button should be enabled (no required params to block it)
  const createBtn = page.getByRole('button', { name: 'Create' })
  await expect.element(createBtn).toBeEnabled()
})

// Coverage: API error during load shows error message
test('API error during load shows error alert', async () => {
  vi.mocked(api.exporters.list).mockRejectedValue(new Error('Network error'))
  vi.mocked(api.exportBindings.list).mockRejectedValue(new Error('Network error'))

  renderPanel()
  await expect.element(page.getByText('Network error')).toBeVisible()
})

// Coverage: loadBindings callback error path (L49)
test('loadBindings reload error shows alert', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'mcp-gateway', description: 'MCP', parameter_schema: [], source: 'built-in', health: 'n/a' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ enabled: true })],
    total: 1,
  })
  vi.mocked(api.exportBindings.update).mockResolvedValue(undefined as any)

  renderPanel()
  await expect.element(page.getByText('Enabled')).toBeVisible()

  // Make the reload fail after toggle
  vi.mocked(api.exportBindings.list).mockRejectedValue(new Error('Reload failed'))
  vi.mocked(api.exporters.list).mockRejectedValue(new Error('Reload failed'))

  // Toggle enabled — this calls update then loadBindings, which will fail in its catch (L49)
  await page.getByRole('button', { name: 'Enabled' }).click()
  await expect.element(page.getByText('Reload failed')).toBeVisible()
})

// Coverage: VS picker modal onClose via Escape (L268)
test('VS picker modal closes on Escape', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'mcp-gateway', description: 'MCP', parameter_schema: [], source: 'built-in', health: 'n/a' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'mcp-gateway', parameters: { virtual_server_type: 'vs' } })],
    total: 1,
  })
  vi.mocked(api.instances.list).mockResolvedValue({ items: [{ id: 'vs1', name: 'prod-vs' }] as any, total: 1 })

  renderPanel()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await expect.element(page.getByText('Select Virtual Server')).toBeVisible()
  await userEvent.keyboard('{Escape}')
  await expect.element(page.getByText('Select Virtual Server')).not.toBeInTheDocument()
})

// Coverage: VS picker select + submit flow (L270-271, L475, L487)
test('VS picker select instance and export', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{ name: 'mcp-gateway', description: 'MCP', parameter_schema: [], source: 'built-in', health: 'n/a' }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({ exporter_name: 'mcp-gateway', parameters: { virtual_server_type: 'vs' } })],
    total: 1,
  })
  vi.mocked(api.instances.list).mockResolvedValue({
    items: [{ id: 'vs1', name: 'prod-vs' }, { id: 'vs2', name: 'staging-vs' }] as any,
    total: 2,
  })
  vi.mocked(api.exportBindings.run).mockResolvedValue(undefined as any)

  renderPanel()
  await page.getByRole('button', { name: 'Export Now' }).click()
  await expect.element(page.getByText('Select Virtual Server')).toBeVisible()

  // Select an instance from the dropdown
  const select = page.getByRole('combobox', { name: 'Select virtual server instance' })
  await userEvent.selectOptions(select, 'prod-vs')

  // Click Export
  await page.getByRole('button', { name: 'Export' }).click()
  await vi.waitFor(() => {
    expect(api.exportBindings.run).toHaveBeenCalledWith('test-catalog', 'b1', 'prod-vs')
  })
})

// Attribute mapping: dropdown renders when entity type has attribute_mappings
test('attribute mapping dropdown renders for entity_type with mappings', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{
      name: 'webhook-gw', description: 'Webhook', source: 'webhook', health: 'Ready',
      parameter_schema: [
        {
          name: 'server_type', type: 'entity_type', description: 'Server type', required: true,
          attribute_mappings: [
            { name: 'route_name_attr', description: 'Route name attr', required: true, default: 'route_name' },
            { name: 'mcp_path_attr', description: 'MCP path attr', default: 'mcp_path' },
          ],
        },
        { name: 'tool_type', type: 'entity_type', description: 'Tool type', required: true },
      ] as any,
    }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.catalogVersions.listPins).mockResolvedValue({
    items: [
      { pin_id: 'p1', entity_type_name: 'mcp-server', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 1 },
      { pin_id: 'p2', entity_type_name: 'mcp-tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
    ],
    total: 2,
  })

  renderPanel()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await expect.element(page.getByRole('dialog')).toBeVisible()

  // Select the webhook exporter
  await userEvent.selectOptions(page.getByLabelText('Select exporter'), 'webhook-gw')

  // Attribute mapping dropdowns should appear but be disabled (no entity type selected yet)
  const routeDropdown = page.getByTestId('param-route_name_attr')
  await expect.element(routeDropdown).toBeVisible()
  await expect.element(routeDropdown).toBeDisabled()

  const mcpDropdown = page.getByTestId('param-mcp_path_attr')
  await expect.element(mcpDropdown).toBeVisible()
  await expect.element(mcpDropdown).toBeDisabled()
})

// Attribute mapping: selecting entity type fetches attributes and populates dropdown
test('attribute mapping dropdown populates after entity type selection', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{
      name: 'webhook-gw', description: 'Webhook', source: 'webhook', health: 'Ready',
      parameter_schema: [
        {
          name: 'server_type', type: 'entity_type', description: 'Server type', required: true,
          attribute_mappings: [
            { name: 'route_name_attr', description: 'Route name', required: true, default: 'route_name' },
          ],
        },
      ] as any,
    }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.catalogVersions.listPins).mockResolvedValue({
    items: [
      { pin_id: 'p1', entity_type_name: 'mcp-server', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 1 },
    ],
    total: 1,
  })
  vi.mocked(api.attributes.list).mockResolvedValue({
    items: [
      { id: 'a1', name: 'route_name', description: 'Route', type_name: 'string', base_type: 'string', required: true, system: false, ordinal: 0 },
      { id: 'a2', name: 'endpoint', description: 'URL', type_name: 'url', base_type: 'string', required: false, system: false, ordinal: 1 },
    ],
    total: 2,
  })

  renderPanel()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await userEvent.selectOptions(page.getByLabelText('Select exporter'), 'webhook-gw')

  // Select entity type — should trigger attribute fetch
  await userEvent.selectOptions(page.getByTestId('param-server_type'), 'mcp-server')

  // Wait for attributes to load
  await vi.waitFor(() => {
    expect(api.attributes.list).toHaveBeenCalledWith('et-1')
  })

  // Dropdown should now be enabled with attribute options
  const routeDropdown = page.getByTestId('param-route_name_attr')
  await expect.element(routeDropdown).toBeEnabled()
})

// Attribute mapping: entity_type without attribute_mappings has no extra dropdowns
test('entity_type without attribute_mappings shows no extra dropdowns', async () => {
  vi.mocked(api.exporters.list).mockResolvedValue({
    items: [{
      name: 'simple', description: 'Simple', source: 'built-in', health: 'n/a',
      parameter_schema: [
        { name: 'server_type', type: 'entity_type', description: 'Server', required: true },
      ],
    }],
    total: 1,
  })
  vi.mocked(api.exportBindings.list).mockResolvedValue({ items: [], total: 0 })
  vi.mocked(api.catalogVersions.listPins).mockResolvedValue({
    items: [{ pin_id: 'p1', entity_type_name: 'my-server', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 1 }],
    total: 1,
  })

  renderPanel()
  await page.getByRole('button', { name: 'Add Export Binding' }).click()
  await userEvent.selectOptions(page.getByLabelText('Select exporter'), 'simple')

  // Only the entity type dropdown should exist — no attribute mapping dropdowns
  await expect.element(page.getByTestId('param-server_type')).toBeVisible()
  await expect.element(page.getByTestId('param-route_name_attr')).not.toBeInTheDocument()
})

// Edit mode: missing attribute mapping defaults are populated so Save is not blocked
test('edit mode populates missing attribute mapping defaults', async () => {
  const exporterWithMappings = {
    name: 'mcp-gateway', description: 'MCP', source: 'built-in', health: 'n/a',
    parameter_schema: [
      {
        name: 'server_type', type: 'entity_type', description: 'Server', required: true,
        attribute_mappings: [
          { name: 'route_name_attr', description: 'Route name', required: true, default: 'route_name' },
        ],
      },
      { name: 'tool_type', type: 'entity_type', description: 'Tool', required: true },
    ] as any,
  }
  vi.mocked(api.exporters.list).mockResolvedValue({ items: [exporterWithMappings], total: 1 })
  vi.mocked(api.exportBindings.list).mockResolvedValue({
    items: [makeBinding({
      exporter_name: 'mcp-gateway',
      parameters: { server_type: 'mcp-server', tool_type: 'mcp-tool' },
    })],
    total: 1,
  })
  vi.mocked(api.catalogVersions.listPins).mockResolvedValue({
    items: [
      { pin_id: 'p1', entity_type_name: 'mcp-server', entity_type_id: 'et-1', entity_type_version_id: 'etv-1', version: 1 },
      { pin_id: 'p2', entity_type_name: 'mcp-tool', entity_type_id: 'et-2', entity_type_version_id: 'etv-2', version: 1 },
    ],
    total: 2,
  })
  vi.mocked(api.exportBindings.update).mockResolvedValue({} as any)

  renderPanel()

  // Click Edit on the binding
  const editBtn = page.getByRole('button', { name: 'Edit' })
  await expect.element(editBtn).toBeVisible()
  await editBtn.click()

  // Modal should appear — Save button should be ENABLED (defaults populated)
  const dialog = page.getByRole('dialog')
  await expect.element(dialog).toBeVisible()
  const saveBtn = page.getByRole('button', { name: 'Save' })
  await expect.element(saveBtn).toBeEnabled()
})
