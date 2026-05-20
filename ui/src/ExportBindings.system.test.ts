// Export Bindings system tests — run against a live deployment.
// Tests: T-34.113 through T-34.117
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   cd ui && npx vitest run --config vitest.system.config.ts src/ExportBindings.system.test.ts

import { test, expect, beforeAll, afterAll, describe } from 'vitest'
import type { Browser, Page } from 'playwright'
import {
  setupBrowser,
  teardownBrowser,
  visible,
  hidden,
  setRole,
  apiCall,
  testName,
  cleanupE2EData,
  cleanupDnsCatalogs,
  getTypeVersionId,
  UI_URL,
} from './test-helpers/system'

let browser: Browser
let pg: Page

let serverETId: string
let toolETId: string
let vsETId: string
let serverETVersionId: string
let toolETVersionId: string
let vsETVersionId: string
let cvId: string

const CATALOG_NAME = 'e2e-export-bindings'
const SERVER_ET_NAME = testName('EB_Server')
const TOOL_ET_NAME = testName('EB_Tool')
const VS_ET_NAME = testName('EB_VirtualServer')
const CV_LABEL = testName('eb-cv')

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  await cleanupDnsCatalogs(CATALOG_NAME, 'e2e-eb-delete-test')
  await cleanupE2EData()

  const stringVersionId = await getTypeVersionId('string')

  // Create server entity type with route_name attribute
  const serverRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: SERVER_ET_NAME,
    description: 'Server type for export binding tests',
  })
  serverETId = serverRes.body.entity_type.id

  // Create tool entity type (before adding containment, since assoc references tool ET)
  const toolRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: TOOL_ET_NAME,
    description: 'Tool type for export binding tests',
  })
  toolETId = toolRes.body.entity_type.id
  toolETVersionId = toolRes.body.version.id

  // Create virtual server entity type
  const vsRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: VS_ET_NAME,
    description: 'Virtual server type for export binding tests',
  })
  vsETId = vsRes.body.entity_type.id
  vsETVersionId = vsRes.body.version.id

  // Add attribute and association — each creates a new version
  await apiCall('POST', `/api/meta/v1/entity-types/${serverETId}/attributes`, {
    name: 'route_name',
    type_definition_version_id: stringVersionId,
    ordinal: 1,
    required: false,
  })

  await apiCall('POST', `/api/meta/v1/entity-types/${serverETId}/associations`, {
    name: 'tools',
    type: 'containment',
    target_entity_type_id: toolETId,
  })

  // Add association from virtual server to tool (required by ValidateSchema)
  await apiCall('POST', `/api/meta/v1/entity-types/${vsETId}/associations`, {
    name: 'tools',
    type: 'directional',
    target_entity_type_id: toolETId,
  })

  // Get the latest versions (after attr+assoc mutations created new versions)
  const vsVersions = await apiCall('GET', `/api/meta/v1/entity-types/${vsETId}/versions`)
  vsETVersionId = vsVersions.body.items[vsVersions.body.items.length - 1].id

  // Get the latest version (after attr+assoc mutations created new versions)
  const serverVersions = await apiCall('GET', `/api/meta/v1/entity-types/${serverETId}/versions`)
  serverETVersionId = serverVersions.body.items[serverVersions.body.items.length - 1].id

  // Create catalog version with pins
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: CV_LABEL })
  cvId = cvRes.body.id

  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: serverETVersionId,
  })
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: toolETVersionId,
  })
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: vsETVersionId,  // latest version with tools association
  })

  // Create catalog
  await apiCall('POST', '/api/data/v1/catalogs', {
    name: CATALOG_NAME,
    description: 'Catalog for export binding tests',
    catalog_version_id: cvId,
  })

  // Create instances for export content verification
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${SERVER_ET_NAME}`, {
    name: 'github',
    description: 'GitHub MCP server',
    attributes: { route_name: 'github-mcp-route' },
  })
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${SERVER_ET_NAME}`, {
    name: 'jira',
    description: 'Jira MCP server',
    attributes: { route_name: 'jira-mcp-route' },
  })
  // Create contained tool instances
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${SERVER_ET_NAME}/github/${TOOL_ET_NAME}`, {
    name: 'list-repos',
    description: 'List GitHub repos',
  })
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${SERVER_ET_NAME}/github/${TOOL_ET_NAME}`, {
    name: 'create-issue',
    description: 'Create GitHub issue',
  })
  // Create virtual-server instance (needed by VS picker)
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${VS_ET_NAME}`, {
    name: 'test-vs',
    description: 'Test virtual server for export',
  })
}, 30000)

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_NAME, 'e2e-eb-delete-test')
  await cleanupE2EData()
  await teardownBrowser(browser)
})

// Helper: navigate to catalog and open Export Plugins tab
async function gotoExportPlugins(page: Page, role: 'RO' | 'RW' | 'Admin' | 'SuperAdmin' = 'Admin', url: string = `${UI_URL}/schema/catalogs/${CATALOG_NAME}`) {
  await page.goto(url)
  await setRole(page, role)
  await visible(page.getByRole('heading', { name: new RegExp(CATALOG_NAME) }), 20000)
  await page.getByRole('tab', { name: 'Export Plugins' }).click()
  await page.waitForTimeout(500)
}

// Helper: ensure a binding exists via API (for tests that need one but don't test creation)
async function ensureBinding() {
  const existing = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings`)
  if ((existing.body.items || []).length === 0) {
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings`, {
      exporter_name: 'mcp-gateway',
      parameters: { server_type: SERVER_ET_NAME, tool_type: TOOL_ET_NAME, virtual_server_type: VS_ET_NAME },
    })
  }
}

// Helper: run export via VS picker modal, returns the download
async function runExportViaPicker(page: Page) {
  const grid = page.locator('table[aria-label="Export bindings"]')
  await grid.getByRole('button', { name: 'Export Now' }).first().click()
  await visible(page.getByRole('dialog'))
  const vsSelect = page.locator('select[aria-label="Select virtual server instance"]')
  await visible(vsSelect)
  await vsSelect.selectOption('test-vs')
  const downloadPromise = page.waitForEvent('download', { timeout: 15000 })
  await page.getByRole('dialog').getByRole('button', { name: 'Export' }).click()
  return downloadPromise
}

describe('Export Bindings E2E', () => {

  // === Phase 1: Empty state + RBAC without bindings ===

  test('T-34.113: Export Plugins tab visible', async () => {
    await gotoExportPlugins(pg)
    await visible(pg.getByRole('tab', { name: 'Export Plugins' }))
  })

  test('T-34.118: Empty state shows no bindings message', async () => {
    // Clean any leftover bindings
    const existing = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings`)
    for (const b of existing.body.items || []) {
      await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings/${b.id}`)
    }
    await gotoExportPlugins(pg)
    await visible(pg.getByText(/No export bindings configured/i))
  })

  test('T-34.119: RO cannot see Add Export Binding button', async () => {
    await gotoExportPlugins(pg, 'RO')
    const addBtn = pg.getByRole('button', { name: 'Add Export Binding' })
    expect(await addBtn.isVisible().catch(() => false)).toBe(false)
  })

  test('T-34.120: RW cannot see Add Export Binding button', async () => {
    await gotoExportPlugins(pg, 'RW')
    const addBtn = pg.getByRole('button', { name: 'Add Export Binding' })
    expect(await addBtn.isVisible().catch(() => false)).toBe(false)
  })

  // === Phase 2: Create binding ===

  test('T-34.114: Add binding via UI', async () => {
    await gotoExportPlugins(pg)

    await pg.getByRole('button', { name: 'Add Export Binding' }).click()
    await visible(pg.getByRole('dialog'))

    // Verify Create button is disabled before selecting exporter
    const createBtn = pg.getByRole('dialog').getByRole('button', { name: 'Create' })
    expect(await createBtn.isDisabled()).toBe(true)

    // Select exporter
    const selectEl = pg.locator('select[aria-label="Select exporter"]')
    await visible(selectEl)
    const options = await selectEl.locator('option').allTextContents()
    const mcpLabel = options.find(o => o.includes('mcp-gateway'))
    expect(mcpLabel).toBeTruthy()
    await selectEl.selectOption({ label: mcpLabel! })

    // Select entity types
    await pg.locator('select#param-server_type').selectOption(SERVER_ET_NAME)
    await pg.locator('select#param-tool_type').selectOption(TOOL_ET_NAME)
    await pg.locator('select#param-virtual_server_type').selectOption(VS_ET_NAME)

    await createBtn.click()

    await hidden(pg.getByRole('dialog'), 20000)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid.getByText('mcp-gateway'))
  })

  // === Phase 3: Binding present — details, edit, toggle ===

  test('T-34.121: Binding row shows exporter name and status never', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    await visible(grid.getByText('mcp-gateway'))
    await visible(grid.getByText('never'))
  })

  test('T-34.122: Edit modal opens with pre-filled parameters', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    await grid.getByRole('button', { name: 'Edit' }).first().click()
    await visible(pg.getByRole('dialog'))

    // Verify parameter values are pre-filled
    const serverSelect = pg.locator('select#param-server_type')
    expect(await serverSelect.inputValue()).toBe(SERVER_ET_NAME)
    const toolSelect = pg.locator('select#param-tool_type')
    expect(await toolSelect.inputValue()).toBe(TOOL_ET_NAME)

    await pg.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    await hidden(pg.getByRole('dialog'))
  })

  test('T-34.123: Edit saves updated parameter', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    await grid.getByRole('button', { name: 'Edit' }).first().click()
    await visible(pg.getByRole('dialog'))

    // Change target_namespace
    const nsInput = pg.locator('input#param-target_namespace')
    if (await nsInput.isVisible()) {
      await nsInput.fill('custom-ns')
      await pg.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
      await hidden(pg.getByRole('dialog'))
      // Verify the change persisted — re-open edit
      await pg.waitForTimeout(500)
      await grid.getByRole('button', { name: 'Edit' }).first().click()
      await visible(pg.getByRole('dialog'))
      expect(await nsInput.inputValue()).toBe('custom-ns')
      await pg.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
      await hidden(pg.getByRole('dialog'))
    } else {
      // No editable text field — just close
      await pg.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
      await hidden(pg.getByRole('dialog'))
    }
  })

  test('T-34.124: Enable/disable toggle works', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)

    // Find and click the toggle
    const toggle = grid.locator('button').filter({ hasText: /Enabled|Disabled/ }).first()
    if (await toggle.isVisible()) {
      const textBefore = await toggle.textContent()
      await toggle.click()
      await pg.waitForTimeout(500)
      const textAfter = await toggle.textContent()
      expect(textAfter).not.toBe(textBefore)
      // Toggle back to enabled for subsequent tests
      if (textAfter?.includes('Disabled')) {
        await toggle.click()
        await pg.waitForTimeout(500)
      }
    }
  })

  test('T-34.125: Disabled binding Export Now is disabled', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)

    // Disable via API for clean state
    const bindings = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings`)
    const binding = bindings.body.items[0]
    await apiCall('PUT', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings/${binding.id}`, { enabled: false })

    await pg.reload()
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('heading', { name: new RegExp(CATALOG_NAME) }))
    await pg.getByRole('tab', { name: 'Export Plugins' }).click()
    await pg.waitForTimeout(500)

    const exportBtn = pg.locator('table[aria-label="Export bindings"]').getByRole('button', { name: 'Export Now' }).first()
    expect(await exportBtn.isDisabled()).toBe(true)

    // Re-enable for subsequent tests
    await apiCall('PUT', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings/${binding.id}`, { enabled: true })
  })

  // === Phase 4: RBAC with binding present ===

  test('T-34.126: RW sees binding list but no Edit/Delete/Toggle', async () => {
    await ensureBinding()
    await gotoExportPlugins(pg, 'RW')
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    await visible(grid.getByText('mcp-gateway'))

    // Should NOT see Edit, Delete, or toggle
    expect(await grid.getByRole('button', { name: 'Edit' }).isVisible().catch(() => false)).toBe(false)
    expect(await grid.getByRole('button', { name: 'Delete' }).isVisible().catch(() => false)).toBe(false)
  })

  test('T-34.127: RO cannot see Export Now, Edit, or Delete', async () => {
    await gotoExportPlugins(pg, 'RO')
    // Binding list should be visible (read-only)
    await pg.waitForTimeout(1000)
    const exportBtn = pg.getByRole('button', { name: 'Export Now' })
    expect(await exportBtn.isVisible().catch(() => false)).toBe(false)
    const editBtn = pg.getByRole('button', { name: 'Edit' })
    expect(await editBtn.isVisible().catch(() => false)).toBe(false)
    const deleteBtn = pg.getByRole('button', { name: 'Delete' })
    expect(await deleteBtn.isVisible().catch(() => false)).toBe(false)
  })

  // === Phase 5: Export YAML content verification ===

  test('T-34.115: Export Now downloads YAML', async () => {
    await ensureBinding()
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)

    const download = await runExportViaPicker(pg)
    expect(download.suggestedFilename()).toContain('.yaml')
  })

  test('T-34.128: Downloaded YAML contains valid CRs with correct apiVersion and labels', async () => {
    await ensureBinding()
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)

    const download = await runExportViaPicker(pg)
    const path = await download.path()
    expect(path).toBeTruthy()
    const fs = await import('fs')
    const content = fs.readFileSync(path!, 'utf8')
    // Verify it's valid multi-document YAML with CRs
    expect(content).toContain('apiVersion: mcp.kuadrant.io/v1alpha1')
    expect(content).toContain('kind:')
    // Verify labels present on generated CRs
    expect(content).toContain('assethub.io/catalog')
    expect(content).toContain('assethub.io/exporter')
    // Verify annotations present
    expect(content).toContain('assethub.io/exported-at')
    // Verify the virtual server name from the picker is in the output
    expect(content).toContain('test-vs')
  })

  test('T-34.129: Last run status updates after export', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    // After previous export, status should show "success" not "never"
    await visible(grid.getByText('success'))
  })

  // === Phase 6: Cross-page consistency ===

  test('T-34.130: Binding visible on operational page', async () => {
    await ensureBinding()
    await pg.goto(`${UI_URL}/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'Admin')
    await visible(pg.getByText(CATALOG_NAME).first(), 20000)
    const exportTab = pg.getByRole('tab', { name: 'Export Plugins' })
    await visible(exportTab)
    await exportTab.click()
    await pg.waitForTimeout(1000)
    await visible(pg.getByText('mcp-gateway'))
  })

  // === Phase 7: Delete edge cases ===

  test('T-34.131: Cancel delete preserves binding', async () => {
    await ensureBinding()
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    const countBefore = await grid.locator('tbody tr').count()

    await grid.getByRole('button', { name: 'Delete' }).first().click()
    await visible(pg.getByRole('dialog'))
    await pg.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    await hidden(pg.getByRole('dialog'))

    const countAfter = await grid.locator('tbody tr').count()
    expect(countAfter).toBe(countBefore)
  })

  test('T-34.132: Escape closes delete modal without deleting', async () => {
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    const countBefore = await grid.locator('tbody tr').count()

    await grid.getByRole('button', { name: 'Delete' }).first().click()
    await visible(pg.getByRole('dialog'))
    await pg.keyboard.press('Escape')
    await hidden(pg.getByRole('dialog'))

    const countAfter = await grid.locator('tbody tr').count()
    expect(countAfter).toBe(countBefore)
  })

  // === Phase 8: Delete binding (existing test) ===

  test('T-34.116: Delete binding via UI', async () => {
    await ensureBinding()
    await gotoExportPlugins(pg)
    const grid = pg.locator('table[aria-label="Export bindings"]')
    await visible(grid)
    const countBefore = await grid.locator('tbody tr').count()
    expect(countBefore).toBeGreaterThan(0)

    await grid.getByRole('button', { name: 'Delete' }).last().click()
    await visible(pg.getByRole('dialog'))
    await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
    await hidden(pg.getByRole('dialog'))

    await pg.waitForTimeout(1000)
    const countAfter = await grid.locator('tbody tr').count()
    expect(countAfter).toBe(countBefore - 1)
  })

  // === Phase 9: Publish with preview ===

  test('T-34.117: Full publish flow with preview', async () => {
    await ensureBinding()

    // Validate via API
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/validate`, {}, 'Admin')

    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('heading', { name: new RegExp(CATALOG_NAME) }), 20000)
    await pg.waitForTimeout(1000)

    const publishBtn = pg.getByRole('button', { name: 'Publish' })
    await visible(publishBtn)
    await publishBtn.click()

    // Preview modal should appear
    await visible(pg.getByRole('dialog'))
    const dialog = pg.getByRole('dialog')
    const previewGrid = dialog.locator('table[aria-label="Preview results"]')
    await visible(previewGrid)

    // Publish in the modal
    const downloadPromise = pg.waitForEvent('download', { timeout: 10000 }).catch(() => null)
    const publishConfirm = dialog.getByRole('button', { name: /Publish/ })
    await visible(publishConfirm)
    await publishConfirm.click()

    await hidden(pg.getByRole('dialog'), 10000)
    await downloadPromise

    // Verify published
    await pg.reload()
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('heading', { name: new RegExp(CATALOG_NAME) }), 20000)
    await visible(pg.getByText('published', { exact: true }))
  }, 60000)

  // === Phase 10: Catalog delete with bindings ===

  test('T-34.133: Catalog delete shows binding count warning', async () => {
    // Create a second catalog for this test (main catalog is now published)
    const deleteCatName = 'e2e-eb-delete-test'
    await cleanupDnsCatalogs(deleteCatName)
    await apiCall('POST', '/api/data/v1/catalogs', {
      name: deleteCatName,
      description: 'Catalog for delete-with-bindings test',
      catalog_version_id: cvId,
    })
    await apiCall('POST', `/api/data/v1/catalogs/${deleteCatName}/export-bindings`, {
      exporter_name: 'mcp-gateway',
      parameters: { server_type: SERVER_ET_NAME, tool_type: TOOL_ET_NAME, virtual_server_type: VS_ET_NAME },
    })

    // Navigate to catalog list page and find the delete button for our catalog
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    await visible(pg.getByText(deleteCatName), 20000)

    // Click Delete on the catalog row
    const row = pg.locator('tr', { hasText: deleteCatName })
    await row.getByRole('button', { name: 'Delete', exact: true }).click()
    await visible(pg.getByRole('dialog'))

    // Verify warning alert mentions export bindings
    await visible(pg.getByRole('heading', { name: /export binding/i }))

    // Confirm delete
    await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
    await hidden(pg.getByRole('dialog'))
    await pg.waitForTimeout(1000)

    // Verify catalog is gone from list
    expect(await pg.getByText(deleteCatName).isVisible().catch(() => false)).toBe(false)
  })
})
