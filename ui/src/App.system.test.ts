// System tests — run against a live deployment (kind cluster).
// Uses Playwright directly to launch a real browser, navigate to the
// deployed UI, and interact with it. No mocks.
//
// Prerequisites:
//   - kind cluster running: ./scripts/kind-deploy.sh up
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   npm run test:system

import { test, expect, beforeAll, afterAll } from 'vitest'
import { chromium, type Browser, type Page, type Locator } from 'playwright'

const UI_URL = 'http://localhost:30000'
const API_URL = 'http://localhost:30080'

let browser: Browser
let pg: Page

// Playwright locator assertion helpers
async function visible(locator: Locator, timeout = 15000) {
  await locator.waitFor({ state: 'visible', timeout })
}

async function hidden(locator: Locator) {
  await locator.waitFor({ state: 'hidden', timeout: 15000 })
}

async function navigateToEntityType(name: string) {
  await navigateToUI()
  await pg.getByPlaceholder('Filter by name').fill(name)
  await visible(pg.getByText(name))
  await pg.getByRole('button', { name }).click()
  await visible(pg.getByRole('heading', { name }))
}

async function navigateToEnumDetail(_enumId: string, enumName: string) {
  await navigateToUI()
  await pg.getByRole('tab', { name: /Enums/i }).click()
  await pg.waitForTimeout(500)
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(enumName))
  await pg.getByRole('button', { name: enumName }).click()
  await visible(pg.getByRole('heading', { name: enumName }))
}

async function apiCall(method: string, path: string, body?: object) {
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-User-Role': 'Admin' },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (method === 'DELETE') return { status: res.status, body: null }
  const text = await res.text()
  try {
    return { status: res.status, body: JSON.parse(text) }
  } catch {
    return { status: res.status, body: text }
  }
}

beforeAll(async () => {
  const health = await fetch(`${API_URL}/healthz`)
  if (!health.ok) throw new Error('API not reachable')

  browser = await chromium.launch({ headless: true })
  pg = await browser.newPage()
})

// Track resources created during tests for cleanup
const createdResources: { type: 'entity-type' | 'enum' | 'catalog-version'; id: string }[] = []

function trackResource(type: 'entity-type' | 'enum' | 'catalog-version', id: string) {
  createdResources.push({ type, id })
}

// Test name prefixes used by system tests — any resource with these prefixes is test data
const TEST_PREFIXES = {
  entityTypes: ['SysTest_', 'FilterTest_', 'DeleteMe_', 'DupTest_', 'DetailTest_', 'AttrTest_', 'Assoc1_', 'Assoc2_', 'VerTest_', 'CopySource_', 'CopyTarget_', 'CopyMultiSrc_', 'CopyMultiTgt_', 'Copied_', 'RenameMe_', 'Renamed_', 'Workflow1_', 'Workflow2_', 'AAA_DelET_', 'ZZZ_DelET_'],
  enums: ['Status_', 'ValTest_', 'DelEnum_', 'AAA_DelEnum_', 'ZZZ_DelEnum_', 'Priority_'],
  catalogVersions: ['v-sys-', 'v-promo-', 'v-del-', 'v-older-', 'v-newer-', 'wf-'],
}

async function cleanupTestData() {
  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'SuperAdmin' }

  // Clean CVs first (may reference entity types)
  try {
    const cvs = await (await fetch(`${API_URL}/api/meta/v1/catalog-versions`, { headers })).json()
    for (const cv of cvs.items || []) {
      if (TEST_PREFIXES.catalogVersions.some(p => cv.version_label.startsWith(p))) {
        await fetch(`${API_URL}/api/meta/v1/catalog-versions/${cv.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }

  // Clean entity types
  try {
    const ets = await (await fetch(`${API_URL}/api/meta/v1/entity-types`, { headers })).json()
    for (const et of ets.items || []) {
      if (TEST_PREFIXES.entityTypes.some(p => et.name.startsWith(p))) {
        await fetch(`${API_URL}/api/meta/v1/entity-types/${et.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }

  // Clean enums
  try {
    const enums = await (await fetch(`${API_URL}/api/meta/v1/enums`, { headers })).json()
    for (const e of enums.items || []) {
      if (TEST_PREFIXES.enums.some(p => e.name.startsWith(p))) {
        await fetch(`${API_URL}/api/meta/v1/enums/${e.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }
}

afterAll(async () => {
  // Clean up all tracked resources (reverse order to handle dependencies)
  for (const r of [...createdResources].reverse()) {
    const path = r.type === 'entity-type' ? `/api/meta/v1/entity-types/${r.id}`
      : r.type === 'enum' ? `/api/meta/v1/enums/${r.id}`
      : `/api/meta/v1/catalog-versions/${r.id}`
    try {
      await fetch(`${API_URL}${path}`, {
        method: 'DELETE',
        headers: { 'X-User-Role': 'SuperAdmin' },
      })
    } catch { /* ignore cleanup errors */ }
  }

  // Final sweep: clean any test data that wasn't tracked (e.g., UI-created resources)
  await cleanupTestData()

  await pg?.close()
  await browser?.close()
})

async function navigateToUI() {
  await pg.goto(UI_URL)
  await pg.waitForLoadState('networkidle')
  await visible(pg.getByText('AI Asset Hub'))
}

// ── Health ──

test('API server is healthy', async () => {
  const res = await apiCall('GET', '/healthz')
  expect((res.body as { status: string }).status).toBe('ok')
})

test('API readiness check passes', async () => {
  const res = await apiCall('GET', '/readyz')
  expect((res.body as { status: string }).status).toBe('ready')
})

// ── UI loads ──

test('UI loads with heading, tabs, and role selector', async () => {
  await navigateToUI()
  await visible(pg.getByRole('button', { name: /Role: Admin/i }))
  await visible(pg.getByRole('tab', { name: /Entity Types/i }))
  await visible(pg.getByRole('tab', { name: /Catalog Versions/i }))
})

test('Entity Types tab shows total count', async () => {
  await navigateToUI()
  // Both tabs render "Total: N" — use first() to get the active tab's total
  await visible(pg.getByText(/Total: \d+/).first())
})

// ── Create entity type ──

test('create entity type end-to-end', async () => {
  const name = `SysTest_${Date.now()}`

  await navigateToUI()
  await pg.getByRole('button', { name: 'Create Entity Type' }).click()
  await visible(pg.getByRole('textbox', { name: /Name/i }))

  await pg.getByRole('textbox', { name: /Name/i }).fill(name)
  await pg.getByRole('textbox', { name: /Description/i }).fill('created by system test')
  await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  // Filter by name to find it even if list is long
  await pg.getByPlaceholder('Filter by name').fill(name)
  await visible(pg.getByText(name))

  // Verify via API
  const res = await apiCall('GET', '/api/meta/v1/entity-types')
  const items = (res.body as { items: { name: string; id: string }[] }).items
  const found = items.find(et => et.name === name)
  expect(found).toBeTruthy()
  trackResource('entity-type', found!.id)

  await apiCall('DELETE', `/api/meta/v1/entity-types/${found!.id}`)
})

// ── Filter ──

test('filter entity types by name', async () => {
  const name = `FilterTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToUI()
  await visible(pg.getByText(name))

  await pg.getByPlaceholder('Filter by name').fill(name)
  await visible(pg.getByText(name))

  await pg.getByPlaceholder('Filter by name').fill('ZZZNONEXIST')
  await visible(pg.getByText('No entity types match the filter.'))

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

// ── Delete ──

test('delete entity type via UI', async () => {
  const name = `DeleteMe_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToUI()
  await pg.getByPlaceholder('Filter by name').fill(name)
  await visible(pg.getByText(name))
  await pg.getByRole('button', { name: 'Delete', exact: true }).click()

  // Confirmation modal now appears — confirm the deletion
  await visible(pg.getByText('Confirm Deletion'))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  await pg.getByPlaceholder('Filter by name').fill('')
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await hidden(pg.getByText(name))

  const check = await apiCall('GET', `/api/meta/v1/entity-types/${etId}`)
  expect(check.status).toBe(404)
})

test('delete entity type targets correct row, not first', async () => {
  const ts = Date.now()
  const nameFirst = `AAA_DelET_${ts}`   // sorts first alphabetically
  const nameSecond = `ZZZ_DelET_${ts}`  // sorts second

  const res1 = await apiCall('POST', '/api/meta/v1/entity-types', { name: nameFirst })
  const res2 = await apiCall('POST', '/api/meta/v1/entity-types', { name: nameSecond })
  const etId1 = (res1.body as { entity_type: { id: string } }).entity_type.id
  const etId2 = (res2.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId1)
  trackResource('entity-type', etId2)

  // Navigate and filter to show both
  await navigateToUI()
  await pg.getByPlaceholder('Filter by name').fill(`_DelET_${ts}`)
  await visible(pg.getByText(nameFirst))
  await visible(pg.getByText(nameSecond))

  // Delete the SECOND one (not first in the list) by targeting its row
  const row2 = pg.getByRole('row').filter({ hasText: nameSecond })
  await row2.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(nameSecond))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  // Verify: second is deleted, first survives
  const check2 = await apiCall('GET', `/api/meta/v1/entity-types/${etId2}`)
  expect(check2.status).toBe(404)
  const check1 = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}`)
  expect(check1.status).toBe(200)

  // Now delete the first one too — verify first dialog is fully dismissed
  await hidden(pg.getByText('Confirm Deletion'))
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await pg.getByPlaceholder('Filter by name').fill(`_DelET_${ts}`)
  await visible(pg.getByText(nameFirst))
  const row1 = pg.getByRole('row').filter({ hasText: nameFirst })
  await row1.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(nameFirst))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  const check1b = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}`)
  expect(check1b.status).toBe(404)
})

// ── Duplicate name error ──

test('duplicate entity type name shows error', async () => {
  const name = `DupTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToUI()
  await pg.getByRole('button', { name: 'Create Entity Type' }).click()
  await pg.getByRole('textbox', { name: /Name/i }).fill(name)
  await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await visible(pg.getByText(/409|conflict|already exists/i))

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

// ── Catalog Versions ──

test('catalog versions tab loads', async () => {
  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await visible(pg.getByRole('button', { name: 'Create Catalog Version' }))
})

test('create catalog version end-to-end', async () => {
  const label = `v-sys-${Date.now()}`

  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await pg.getByRole('button', { name: 'Create Catalog Version' }).click()
  await pg.getByPlaceholder('e.g. v1.0').fill(label)
  await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  await visible(pg.getByText(label))

  // Look up the created CV by label for cleanup tracking
  const cvs = await apiCall('GET', '/api/meta/v1/catalog-versions')
  const found = (cvs.body as { items: { id: string; version_label: string }[] }).items.find(cv => cv.version_label === label)
  if (found) trackResource('catalog-version', found.id)
})

test('promote catalog version', async () => {
  const label = `v-promo-${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: label })
  const cvId = (res.body as { id: string }).id
  trackResource('catalog-version', cvId)

  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await visible(pg.getByText(label))

  await pg.getByRole('button', { name: 'Promote' }).first().click()

  // Verify via API that stage changed to testing
  const check = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId}`)
  expect((check.body as { lifecycle_stage: string }).lifecycle_stage).toBe('testing')
})

// ── Role switching ──

test('switching to RO hides admin controls', async () => {
  await navigateToUI()
  await visible(pg.getByRole('button', { name: 'Create Entity Type' }))

  await pg.getByRole('button', { name: /Role: Admin/i }).click()
  await pg.getByRole('option', { name: 'RO' }).click()

  await visible(pg.getByRole('button', { name: /Role: RO/i }))
  await hidden(pg.getByRole('button', { name: 'Create Entity Type' }))
})

test('RO hides catalog version admin controls', async () => {
  await navigateToUI()
  await pg.getByRole('button', { name: /Role: Admin/i }).click()
  await pg.getByRole('option', { name: 'RO' }).click()

  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await hidden(pg.getByRole('button', { name: 'Create Catalog Version' }))
})

// ── Entity Type Detail Page ──

test('navigate to entity type detail page', async () => {
  const name = `DetailTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToUI()
  await pg.getByPlaceholder('Filter by name').fill(name)
  await visible(pg.getByText(name))

  // Click the entity type name link
  await pg.getByRole('button', { name }).click()

  // Should see the detail page with overview
  await visible(pg.getByRole('heading', { name }))
  await visible(pg.getByRole('tab', { name: 'Overview' }))
  await visible(pg.getByRole('tab', { name: 'Attributes' }))
  await visible(pg.getByRole('tab', { name: 'Associations' }))
  await visible(pg.getByRole('tab', { name: 'Version History' }))

  // Back link exists
  await visible(pg.getByRole('button', { name: /Back to Entity Types/i }))

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

test('add and remove attribute on entity type', async () => {
  const name = `AttrTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToEntityType(name)

  // Go to attributes tab
  await pg.getByRole('tab', { name: /Attributes/i }).click()
  await visible(pg.getByText('No attributes defined yet.'))

  // Add an attribute
  await pg.getByRole('button', { name: 'Add Attribute' }).click()
  await pg.getByRole('textbox', { name: /Name/i }).fill('hostname')
  await pg.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  // Attribute should appear
  await visible(pg.getByText('hostname'))

  // Verify via API
  const attrs = await apiCall('GET', `/api/meta/v1/entity-types/${etId}/attributes`)
  const items = (attrs.body as { items: { name: string }[] }).items
  expect(items.some(a => a.name === 'hostname')).toBe(true)

  // Remove the attribute
  await pg.getByRole('button', { name: 'Remove' }).click()
  await visible(pg.getByText('No attributes defined yet.'))

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

test('add association between entity types', async () => {
  const ts = Date.now()
  const name1 = `Assoc1_${ts}`
  const name2 = `Assoc2_${ts}`
  const res1 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name1 })
  const res2 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name2 })
  const etId1 = (res1.body as { entity_type: { id: string } }).entity_type.id
  const etId2 = (res2.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId1)
  trackResource('entity-type', etId2)

  // Create association via API (PatternFly Select dropdowns are unreliable in E2E)
  const targetRole = `role_${ts}`
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/associations`, {
    target_entity_type_id: etId2, type: 'directional', name: `ref_${ts}`, target_role: targetRole,
  })

  // Navigate and verify it shows in the UI
  await navigateToEntityType(name1)
  await pg.getByRole('tab', { name: /Associations/i }).click()
  await visible(pg.getByText(name2))
  await visible(pg.getByText('references'))

  // Remove via UI — target the specific association row by target name and target role.
  // Target + target role is a unique combination per entity type.
  // TODO: Add tests for uniqueness enforcement once the API rejects duplicate target+target_role.
  const assocRow = pg.getByRole('row').filter({ hasText: name2 }).filter({ hasText: targetRole })
  await assocRow.getByRole('button', { name: 'Remove' }).click()

  // Wait for the backend delete to complete before verifying via API
  await pg.waitForTimeout(300)

  // Verify removal via API
  const assocs = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}/associations`)
  const items = (assocs.body as { items: { target_entity_type_id: string }[] }).items
  expect(items.length).toBe(0)

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId1}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId2}`)
})

test('rename entity type and navigate back shows new name in list', async () => {
  const originalName = `RenameMe_${Date.now()}`
  const newName = `Renamed_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name: originalName })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  // Navigate to entity type detail page
  await navigateToEntityType(originalName)

  // Rename via Rename link on the overview tab
  await visible(pg.getByText('Rename', { exact: true }))
  await pg.getByText('Rename', { exact: true }).click()
  await visible(pg.getByText('Rename Entity Type'))
  await pg.getByRole('textbox', { name: /New Name/i }).clear()
  await pg.getByRole('textbox', { name: /New Name/i }).fill(newName)
  await pg.getByRole('dialog').getByRole('button', { name: 'Rename' }).click()

  // Wait for rename to complete — heading should update
  await visible(pg.getByRole('heading', { name: newName }))

  // Navigate back to entity types list
  await pg.getByRole('button', { name: /Back to Entity Types/i }).click()

  // The list should show the new name, not the old one
  await pg.getByPlaceholder('Filter by name').fill(newName)
  await visible(pg.getByText(newName))

  // Old name should not appear
  await pg.getByPlaceholder('Filter by name').fill(originalName)
  await visible(pg.getByText('No entity types match the filter.'))

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

test('copy attributes from multi-version entity type works correctly', async () => {
  const ts = Date.now()

  // Create source entity type and add attributes across multiple versions
  const sourceName = `CopyMultiSrc_${ts}`
  const sourceRes = await apiCall('POST', '/api/meta/v1/entity-types', { name: sourceName })
  const sourceEtId = (sourceRes.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', sourceEtId)

  // Add attribute to source (creates V2)
  await apiCall('POST', `/api/meta/v1/entity-types/${sourceEtId}/attributes`, { name: 'added_later', type: 'string' })

  // Verify source is now at V2 with the attribute
  const sourceAttrs = await apiCall('GET', `/api/meta/v1/entity-types/${sourceEtId}/attributes`)
  const sourceItems = (sourceAttrs.body as { items: { name: string }[] }).items
  expect(sourceItems.some(a => a.name === 'added_later')).toBe(true)

  // Create target entity type
  const targetName = `CopyMultiTgt_${ts}`
  const targetRes = await apiCall('POST', '/api/meta/v1/entity-types', { name: targetName })
  const targetEtId = (targetRes.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', targetEtId)

  // Navigate to target, open copy-from modal, select source, copy the V2 attribute
  await navigateToEntityType(targetName)
  await pg.getByRole('tab', { name: /Attributes/i }).click()
  await visible(pg.getByText('No attributes defined yet.'))

  await pg.getByRole('button', { name: 'Copy from...' }).click()
  await visible(pg.getByText('Copy Attributes from Another Type'))

  await pg.getByRole('button', { name: 'Select source type' }).click()
  await pg.getByText(sourceName).click()

  // The attribute added in V2 should be visible
  await visible(pg.getByText('added_later'))

  // Select it and copy
  const attrRow = pg.getByRole('row').filter({ hasText: 'added_later' })
  await attrRow.locator('input[type="checkbox"]').click()
  await pg.getByRole('button', { name: 'Copy Selected' }).click()

  // Should succeed — no error shown
  // Wait for the modal to close (indicates success)
  await hidden(pg.getByText('Copy Attributes from Another Type'))

  // Attribute should appear in the target's attribute list
  await visible(pg.getByRole('tab', { name: /Attributes/i }))
  await visible(pg.getByText('added_later'))

  // Verify via API
  const targetAttrs = await apiCall('GET', `/api/meta/v1/entity-types/${targetEtId}/attributes`)
  const targetItems = (targetAttrs.body as { items: { name: string }[] }).items
  expect(targetItems.some(a => a.name === 'added_later')).toBe(true)

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/entity-types/${targetEtId}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${sourceEtId}`)
})

test('copy attributes picker shows enum name for enum-type attributes', async () => {
  const ts = Date.now()
  // Create an enum
  const enumName = `Priority_${ts}`
  const enumRes = await apiCall('POST', '/api/meta/v1/enums', { name: enumName, values: ['high', 'medium', 'low'] })
  const enumId = (enumRes.body as { id: string }).id
  trackResource('enum', enumId)

  // Create source entity type with an enum attribute
  const sourceName = `CopySource_${ts}`
  const sourceRes = await apiCall('POST', '/api/meta/v1/entity-types', { name: sourceName })
  const sourceEtId = (sourceRes.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', sourceEtId)

  await apiCall('POST', `/api/meta/v1/entity-types/${sourceEtId}/attributes`, {
    name: 'task_priority', type: 'enum', enum_id: enumId,
  })

  // Create target entity type
  const targetName = `CopyTarget_${ts}`
  const targetRes = await apiCall('POST', '/api/meta/v1/entity-types', { name: targetName })
  const targetEtId = (targetRes.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', targetEtId)

  // Navigate to target entity type detail
  await navigateToEntityType(targetName)
  await pg.getByRole('tab', { name: /Attributes/i }).click()
  await visible(pg.getByText('No attributes defined yet.'))

  // Open copy-from modal
  await pg.getByRole('button', { name: 'Copy from...' }).click()
  await visible(pg.getByText('Copy Attributes from Another Type'))

  // Select source entity type from the dropdown
  await pg.getByRole('button', { name: 'Select source type' }).click()
  await pg.getByText(sourceName).click()

  // The enum attribute should show "enum (Priority_<ts>)" — the enum name, not just "enum"
  await visible(pg.getByText(`enum (${enumName})`))

  // Close modal
  await pg.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/entity-types/${targetEtId}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${sourceEtId}`)
  await apiCall('DELETE', `/api/meta/v1/enums/${enumId}`)
})

test('version history shows versions and diff', async () => {
  const name = `VerTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  // Add an attribute via API to create version 2
  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/attributes`, {
    name: 'cpu_count', type: 'number',
  })

  await navigateToEntityType(name)

  // Go to version history tab
  await pg.getByRole('tab', { name: /Version History/i }).click()
  await visible(pg.getByText('V1'))
  await visible(pg.getByText('V2'))

  // Compare versions
  await pg.getByRole('spinbutton', { name: 'From version' }).fill('1')
  await pg.getByRole('spinbutton', { name: 'To version' }).fill('2')
  await pg.getByRole('button', { name: 'Compare' }).click()

  // Diff should show cpu_count was added
  await visible(pg.getByText('cpu_count'))

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
})

test('copy entity type from detail page', async () => {
  const name = `CopySource_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId)

  await navigateToEntityType(name)

  // Click Copy
  await pg.getByRole('button', { name: 'Copy' }).click()
  await visible(pg.getByText('Copy Entity Type'))

  const copyName = `Copied_${Date.now()}`
  await pg.getByRole('textbox', { name: /New Name/i }).fill(copyName)
  await pg.getByRole('dialog').getByRole('button', { name: 'Copy' }).click()

  // Should navigate back to list
  await visible(pg.getByText('Entity Types'))

  // Verify copy exists via API
  const list = await apiCall('GET', '/api/meta/v1/entity-types')
  const items = (list.body as { items: { name: string; id: string }[] }).items
  const copy = items.find(et => et.name === copyName)
  expect(copy).toBeTruthy()
  trackResource('entity-type', copy!.id)

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${copy!.id}`)
})

// ── Enum Management ──

test('enums tab shows and create enum', async () => {
  await navigateToUI()
  await pg.getByRole('tab', { name: /Enums/i }).click()
  await visible(pg.getByRole('button', { name: 'Refresh' }))

  // Create enum
  await pg.getByRole('button', { name: 'Create Enum' }).click()
  await visible(pg.getByRole('dialog'))

  const enumName = `Status_${Date.now()}`
  await pg.getByRole('textbox', { name: /Name/i }).fill(enumName)
  await pg.getByPlaceholder('e.g. active, inactive, pending').fill('active, inactive')
  await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

  // Should appear in the list
  await visible(pg.getByText(enumName))

  // Verify via API
  const enums = await apiCall('GET', '/api/meta/v1/enums')
  const items = (enums.body as { items: { name: string; id: string }[] }).items
  const found = items.find(e => e.name === enumName)
  expect(found).toBeTruthy()
  trackResource('enum', found!.id)

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/enums/${found!.id}`)
})

test('navigate to enum detail and manage values', async () => {
  // Create enum with values via API
  const enumName = `ValTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/enums', { name: enumName, values: ['alpha', 'beta'] })
  const enumId = (res.body as { id: string }).id
  trackResource('enum', enumId)

  await navigateToEnumDetail(enumId, enumName)

  // Values should be visible
  await visible(pg.getByText('alpha'))
  await visible(pg.getByText('beta'))

  // Add a value
  await pg.getByRole('button', { name: 'Add Value' }).click()
  await pg.getByRole('textbox', { name: /Value/i }).fill('gamma')
  await pg.getByRole('dialog').getByRole('button', { name: 'Add' }).click()
  await visible(pg.getByText('gamma'))

  // Verify via API
  const vals = await apiCall('GET', `/api/meta/v1/enums/${enumId}/values`)
  const items = (vals.body as { items: { value: string }[] }).items
  expect(items.some(v => v.value === 'gamma')).toBe(true)

  // Remove a value
  await pg.getByRole('button', { name: 'Remove' }).first().click()

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/enums/${enumId}`)
})

test('delete enum with confirmation', async () => {
  const enumName = `DelEnum_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/enums', { name: enumName })
  const enumId = (res.body as { id: string }).id
  trackResource('enum', enumId)

  await navigateToUI()
  await pg.getByRole('tab', { name: /Enums/i }).click()
  // Wait for enum list to load then refresh to ensure our new enum appears
  await pg.waitForTimeout(1000)
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(enumName))

  // Click Delete on the row containing our enum name
  const enumRow = pg.getByRole('row').filter({ hasText: enumName })
  await enumRow.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(enumName))

  // Confirm and wait for dialog to close (confirms backend processed the delete)
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  // Verify via API
  const check = await apiCall('GET', `/api/meta/v1/enums/${enumId}`)
  expect(check.status).toBe(404)
})

test('delete enum targets correct row, not first', async () => {
  const ts = Date.now()
  const nameFirst = `AAA_DelEnum_${ts}`   // sorts first alphabetically
  const nameSecond = `ZZZ_DelEnum_${ts}`  // sorts second

  const res1 = await apiCall('POST', '/api/meta/v1/enums', { name: nameFirst })
  const res2 = await apiCall('POST', '/api/meta/v1/enums', { name: nameSecond })
  const enumId1 = (res1.body as { id: string }).id
  const enumId2 = (res2.body as { id: string }).id
  trackResource('enum', enumId1)
  trackResource('enum', enumId2)

  await navigateToUI()
  await pg.getByRole('tab', { name: /Enums/i }).click()
  await pg.waitForTimeout(500)
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(nameFirst))
  await visible(pg.getByText(nameSecond))

  // Delete the SECOND one (not first in the list) by targeting its row
  const row2 = pg.getByRole('row').filter({ hasText: nameSecond })
  await row2.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(nameSecond))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  // Verify: second is deleted, first survives
  const check2 = await apiCall('GET', `/api/meta/v1/enums/${enumId2}`)
  expect(check2.status).toBe(404)
  const check1 = await apiCall('GET', `/api/meta/v1/enums/${enumId1}`)
  expect(check1.status).toBe(200)

  // Now delete the first one too — verify first dialog is fully dismissed
  await hidden(pg.getByText('Confirm Deletion'))
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(nameFirst))
  const row1 = pg.getByRole('row').filter({ hasText: nameFirst })
  await row1.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(nameFirst))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  const check1b = await apiCall('GET', `/api/meta/v1/enums/${enumId1}`)
  expect(check1b.status).toBe(404)
})

// ── Catalog Version Delete ──

test('delete catalog version via UI', async () => {
  const label = `v-del-${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: label })
  const cvId = (res.body as { id: string }).id
  trackResource('catalog-version', cvId)

  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await visible(pg.getByText(label))

  // Click Delete on the row containing our catalog version label
  const cvRow = pg.getByRole('row').filter({ hasText: label })
  await cvRow.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(label))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  // Verify deleted via API
  const check = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId}`)
  expect(check.status).toBe(404)
})

test('delete catalog version targets correct row, not first', async () => {
  const ts = Date.now()
  // CVs sort by created_at DESC, so create the "survivor" first (it will be second in the list)
  const labelOlder = `v-older-${ts}`
  const res1 = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: labelOlder })
  const cvId1 = (res1.body as { id: string }).id
  trackResource('catalog-version', cvId1)

  // WORKAROUND: Wait to ensure different created_at timestamps so sort order is deterministic.
  // CVs sort by created_at DESC — if both have the same timestamp, order is non-deterministic.
  // TODO: Remove this delay once the backend enforces unique CV timestamps.
  await new Promise(r => setTimeout(r, 1100))

  const labelNewer = `v-newer-${ts}`
  const res2 = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: labelNewer })
  const cvId2 = (res2.body as { id: string }).id
  trackResource('catalog-version', cvId2)

  // Verify timestamps are actually different (newer should have later created_at)
  const checkOlderData = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId1}`)
  const checkNewerData = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId2}`)
  const olderTime = (checkOlderData.body as { created_at: string }).created_at
  const newerTime = (checkNewerData.body as { created_at: string }).created_at
  expect(new Date(newerTime).getTime()).toBeGreaterThan(new Date(olderTime).getTime())

  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await visible(pg.getByText(labelOlder))
  await visible(pg.getByText(labelNewer))

  // Delete the OLDER one (second in the list since CVs sort by created_at DESC)
  const rowOlder = pg.getByRole('row').filter({ hasText: labelOlder })
  await rowOlder.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  await visible(pg.getByRole('dialog').getByText(labelOlder))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
  await hidden(pg.getByText('Confirm Deletion'))

  // Verify: older is deleted, newer survives
  const checkOlder = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId1}`)
  expect(checkOlder.status).toBe(404)
  const checkNewer = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId2}`)
  expect(checkNewer.status).toBe(200)

  // Now delete the newer one too — verify first dialog is fully dismissed
  await hidden(pg.getByText('Confirm Deletion'))
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(labelNewer))

  const rowNewer = pg.getByRole('row').filter({ hasText: labelNewer })
  await rowNewer.getByRole('button', { name: 'Delete' }).click()
  await visible(pg.getByText('Confirm Deletion'))
  // Verify the dialog mentions the right CV
  await visible(pg.getByRole('dialog').getByText(labelNewer))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  // Wait for dialog to close, confirming delete was processed
  await hidden(pg.getByText('Confirm Deletion'))

  const checkNewerB = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId2}`)
  expect(checkNewerB.status).toBe(404)
})

// ── Full Workflow ──

test('full workflow: create entity type → add attributes → add association → create catalog version', async () => {
  // Create two entity types
  const name1 = `Workflow1_${Date.now()}`
  const name2 = `Workflow2_${Date.now()}`
  const res1 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name1 })
  const res2 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name2 })
  const etId1 = (res1.body as { entity_type: { id: string } }).entity_type.id
  const etId2 = (res2.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId1)
  trackResource('entity-type', etId2)

  // Add attributes to entity type 1 via API
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/attributes`, { name: 'hostname', type: 'string' })
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/attributes`, { name: 'memory_gb', type: 'number' })

  // Add association via API
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/associations`, {
    target_entity_type_id: etId2, type: 'directional', name: `ref_${Date.now()}`,
  })

  await navigateToEntityType(name1)

  // Check attributes
  await pg.getByRole('tab', { name: /Attributes/i }).click()
  await visible(pg.getByText('hostname'))
  await visible(pg.getByText('memory_gb'))

  // Check associations
  await pg.getByRole('tab', { name: /Associations/i }).click()
  await visible(pg.getByText(name2))

  // Check version history (should have multiple versions from attribute adds)
  await pg.getByRole('tab', { name: /Version History/i }).click()
  await visible(pg.getByText('V1'))

  // Get latest version for catalog version pin
  const versions = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}/versions`)
  const vItems = (versions.body as { items: { id: string; version: number }[] }).items
  const latest = vItems.reduce((a, b) => a.version > b.version ? a : b)

  // Create catalog version pinning this entity type version
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: `wf-${Date.now()}`,
    pins: [{ entity_type_version_id: latest.id }],
  })
  expect(cvRes.status).toBe(201)

  // Clean up
  const cvId = (cvRes.body as { id: string }).id
  trackResource('catalog-version', cvId)
  await apiCall('DELETE', `/api/meta/v1/catalog-versions/${cvId}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId1}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId2}`)
})

// Association cardinality: create with non-default values, verify in API and UI
test('Association cardinality is stored and displayed', async () => {
  const ts = Date.now()
  const name1 = `CardSrc_${ts}`
  const name2 = `CardTgt_${ts}`

  // Create two entity types
  const res1 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name1 })
  const res2 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name2 })
  const etId1 = (res1.body as { entity_type: { id: string } }).entity_type.id
  const etId2 = (res2.body as { entity_type: { id: string } }).entity_type.id
  trackResource('entity-type', etId1)
  trackResource('entity-type', etId2)

  // Create association with non-default cardinality
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/associations`, {
    target_entity_type_id: etId2, type: 'containment',
    name: `tools_${ts}`,
    source_role: 'server', target_role: 'tool',
    source_cardinality: '1', target_cardinality: '1..n',
  })

  // Verify via API — cardinality in list response
  const assocRes = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}/associations`)
  const assocItems = (assocRes.body as { items: { source_cardinality: string; target_cardinality: string; direction: string }[] }).items
  const outgoing = assocItems.find(a => a.direction === 'outgoing')
  expect(outgoing).toBeDefined()
  expect(outgoing!.source_cardinality).toBe('1')
  expect(outgoing!.target_cardinality).toBe('1..n')

  // Verify in UI
  await navigateToEntityType(name1)
  await pg.getByRole('tab', { name: /Associations/i }).click()
  await visible(pg.getByText('1 → 1..n'))

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId1}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId2}`)
})
