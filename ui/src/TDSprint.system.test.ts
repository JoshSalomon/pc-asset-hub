import { describe, test, beforeAll, afterAll, expect } from 'vitest'
import type { Browser, Page } from 'playwright'
import {
  setupBrowser, teardownBrowser, visible, setRole,
  apiCall, UI_URL, cleanupDnsCatalogs,
} from './test-helpers/system'

let browser: Browser
let pg: Page
const CATALOG_NAME = 'e2e-td-sprint'

beforeAll(async () => {
  ;({ browser, page: pg } = await setupBrowser())

  await cleanupDnsCatalogs(CATALOG_NAME)

  const cvRes = await apiCall('GET', '/api/meta/v1/catalog-versions')
  let cv = null
  for (const c of cvRes.body.items || []) {
    const pins = await apiCall('GET', `/api/meta/v1/catalog-versions/${c.id}/pins`)
    const hasServer = (pins.body.items || []).some((p: { entity_type_name: string }) => p.entity_type_name === 'mcp-server')
    if (hasServer) { cv = c; break }
  }
  if (!cv) throw new Error('No catalog version with mcp-server pin found')

  await apiCall('POST', '/api/data/v1/catalogs', {
    name: CATALOG_NAME,
    catalog_version_id: cv.id,
  })

  const etRes = await apiCall('GET', '/api/meta/v1/entity-types')
  const serverET = etRes.body.items?.find((e: { name: string }) => e.name === 'mcp-server')
  const toolET = etRes.body.items?.find((e: { name: string }) => e.name === 'mcp-tool')
  if (!serverET || !toolET) throw new Error('mcp-server or mcp-tool entity type not found')

  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/mcp-server`, {
    name: 'test-server', description: 'test server',
    attributes: { endpoint: 'https://test.example.com/mcp' },
  })
  const serverList = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/mcp-server`)
  const server = serverList.body.items?.[0]
  if (!server) throw new Error('Failed to create test server')

  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/mcp-server/${server.id}/mcp-tool`, {
    name: 'test-tool', description: 'test tool',
  })

  // Validate — should pass (server has endpoint + contained tool)
  const valResult = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/validate`)
  if (valResult.body.status !== 'valid') {
    const errors = valResult.body.errors?.map((e: { violation: string }) => e.violation).join('; ')
    throw new Error(`Validation failed: ${errors}`)
  }

  const bindingRes = await apiCall('GET', `/api/data/v1/exporters`)
  const exporterName = bindingRes.body.items?.[0]?.name || 'mcp-gateway'
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/export-bindings`, {
    exporter_name: exporterName,
    parameters: {
      server_type: 'mcp-server',
      tool_type: 'mcp-tool',
      virtual_server_type: 'virtual-server',
      target_namespace: 'test-ns',
    },
  })

  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/publish`, undefined, 'SuperAdmin')

  // Create an orphaned mcp-tool AFTER publishing (SuperAdmin can mutate published catalogs)
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/mcp-tool`, {
    name: 'orphan-tool', description: 'orphaned tool without parent',
  }, 'SuperAdmin')
}, 60000)

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_NAME)
  await teardownBrowser(browser)
})

// ============================================================
// Tests 1-2: Security — TD-148 Unpublish button visibility
// ============================================================

describe('TD-148: Unpublish button on published catalog', () => {
  test('1. Admin does NOT see Unpublish button on published catalog', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')
    await pg.waitForLoadState('networkidle')

    await visible(pg.getByText(CATALOG_NAME).first(), 10000)
    await visible(pg.getByText('published').first())
    expect(await pg.getByRole('button', { name: 'Unpublish' }).isVisible()).toBe(false)

    await visible(pg.getByRole('button', { name: 'Validate' }))
    await visible(pg.getByRole('button', { name: 'Copy' }))
    await visible(pg.getByRole('button', { name: 'Export' }))
  })

  test('2. SuperAdmin sees Unpublish button and can unpublish', async () => {
    await setRole(pg, 'SuperAdmin')
    await visible(pg.getByRole('button', { name: 'Unpublish' }))

    await pg.getByRole('button', { name: 'Unpublish' }).click()
    await pg.waitForLoadState('networkidle')

    // After unpublish, Unpublish button should be gone
    await pg.waitForTimeout(1000)
    expect(await pg.getByRole('button', { name: 'Unpublish' }).isVisible()).toBe(false)

    // Re-validate and re-publish for subsequent tests
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/validate`, undefined, 'SuperAdmin')
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/publish`, undefined, 'SuperAdmin')
  })
})

// ============================================================
// Tests 3-4: Security — TD-150 Export binding parameter filtering
// ============================================================

describe('TD-150: Export binding parameters visibility', () => {
  test('3. RO/RW cannot see binding parameter values', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'RO')
    await pg.waitForLoadState('networkidle')

    await pg.getByRole('tab', { name: 'Export Plugins' }).click()
    await visible(pg.getByText('mcp-gateway'))

    expect(await pg.getByText('test-ns').isVisible()).toBe(false)
    expect(await pg.getByText('target_namespace').isVisible()).toBe(false)
  })

  test('4. Admin sees binding parameter values', async () => {
    await setRole(pg, 'Admin')
    await pg.waitForTimeout(500)

    await visible(pg.getByText('mcp-gateway'))
    await visible(pg.getByText('target_namespace'))
  })
})

// ============================================================
// Test 5: UX — TD-79 Add Pin auto-selects latest version
// ============================================================

describe('TD-79: Add Pin version auto-select', () => {
  test('5. Add Pin modal auto-selects latest version after entity type selection', async () => {
    // Use the "test" CV which has no pins — so Add Pin modal shows all entity types
    const cvRes = await apiCall('GET', '/api/meta/v1/catalog-versions')
    const testCV = cvRes.body.items?.find((c: { version_label: string }) => c.version_label === 'test')
    if (!testCV) { console.log('SKIP: no "test" CV available'); return }

    await pg.goto(`${UI_URL}/schema/catalog-versions/${testCV.id}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')
    await pg.waitForLoadState('networkidle')

    await pg.getByRole('tab', { name: /Bill of Materials/i }).click()
    await visible(pg.getByRole('button', { name: 'Add Pin' }))
    await pg.getByRole('button', { name: 'Add Pin' }).click()

    const modal = pg.getByRole('dialog')
    await visible(modal)

    // Select entity type via the dropdown toggle
    const etToggle = modal.getByText('Select entity type...').first()
    await visible(etToggle, 5000)
    await etToggle.click()
    const etOption = pg.getByRole('option').first()
    await visible(etOption)
    await etOption.click()

    // After entity type selection, the version dropdown should auto-select (not show "Select version...")
    await pg.waitForTimeout(1000)
    expect(await modal.getByText('Select version...').isVisible()).toBe(false)

    await pg.keyboard.press('Escape')
  }, 45000)
})

// ============================================================
// Test 6: UX — TD-33 "Contained by" shows parent name instantly
// ============================================================

describe('TD-33: Contained by parent name', () => {
  test('6. Schema page: contained instance shows parent name, not UUID', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'SuperAdmin')
    await pg.waitForLoadState('networkidle')

    await visible(pg.getByText(CATALOG_NAME).first(), 10000)

    // Navigate to mcp-tool tab to see contained instances
    const toolTab = pg.getByRole('tab', { name: 'mcp-tool' })
    await visible(toolTab)
    await toolTab.click()
    await pg.waitForTimeout(2000)

    // Click Details on test-tool
    const detailsBtn = pg.getByRole('row').filter({ hasText: 'test-tool' }).getByRole('button', { name: 'Details' })
    await visible(detailsBtn, 10000)
    await detailsBtn.click()
    await pg.waitForTimeout(1000)

    // "Contained by" is inside the details pane which is on the currently visible tab
    // Use a locator scoped to the visible tab panel
    const containedByText = pg.locator('p:has-text("Contained by: test-server"):visible')
    await visible(containedByText.first(), 10000)
  })
})

// ============================================================
// Test 7: UX — TD-142 Add Child pre-selects single child type
// ============================================================

describe('TD-142: Add Child pre-select', () => {
  test('7. Operational page: Add Child pre-selects child type when only one containment type', async () => {
    await pg.goto(`${UI_URL}/catalogs/${CATALOG_NAME}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'SuperAdmin')
    await pg.waitForLoadState('networkidle')
    await pg.waitForTimeout(2000)

    // The tree should show mcp-server group — expand it and click test-server
    const serverGroup = pg.getByText(/mcp-server/i).first()
    await visible(serverGroup, 15000)
    await serverGroup.click()
    await pg.waitForTimeout(500)

    // Click the server instance node
    const serverNode = pg.getByText('test-server').first()
    await visible(serverNode, 10000)
    await serverNode.click()
    await pg.waitForTimeout(1000)

    // Wait for Add Child button in the detail panel
    const addChildBtn = pg.getByRole('button', { name: 'Add Child' })
    await visible(addChildBtn, 10000)
    await addChildBtn.click()

    const modal = pg.getByRole('dialog')
    await visible(modal)

    // The child type dropdown should show mcp-tool pre-selected (only one containment target)
    await visible(modal.getByText('mcp-tool').first())

    await pg.keyboard.press('Escape')
  }, 60000)
})

// ============================================================
// Test 8: UX — TD-146 Orphaned containment targets visual marker
// ============================================================

describe('TD-146: Orphaned containment target warning', () => {
  test('8. Operational page: orphaned instances at root show warning icon', async () => {
    await pg.goto(`${UI_URL}/catalogs/${CATALOG_NAME}`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')
    await pg.waitForLoadState('networkidle')

    // The orphan-tool instance is mcp-tool type without a parent, so the
    // mcp-tool group header at root should show a ⚠ warning icon
    await visible(pg.locator('span[title*="containment target"]').first(), 15000)
  })
})

// ============================================================
// Test 9: UX — TD-137 Types tab name filter
// ============================================================

describe('TD-137: Types tab name filter', () => {
  test('9. Schema page: Types tab search box filters by name', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: 'Types', exact: true }).click()
    await pg.waitForLoadState('networkidle')

    await visible(pg.getByRole('grid'))

    const searchInput = pg.getByPlaceholder('Filter types by name')
    await visible(searchInput)

    await searchInput.fill('integ')
    await pg.waitForTimeout(500)

    await visible(pg.getByRole('gridcell', { name: 'integer' }).first())
    expect(await pg.getByRole('row').filter({ hasText: 'string' }).first().isVisible()).toBe(false)

    await searchInput.fill('')
    await pg.waitForTimeout(500)
    await visible(pg.getByRole('gridcell', { name: 'string' }).first())
  })
})
