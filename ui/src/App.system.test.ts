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

afterAll(async () => {
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

  await apiCall('DELETE', `/api/meta/v1/entity-types/${found!.id}`)
})

// ── Filter ──

test('filter entity types by name', async () => {
  const name = `FilterTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id

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

// ── Duplicate name error ──

test('duplicate entity type name shows error', async () => {
  const name = `DupTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id

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
})

test('promote catalog version', async () => {
  const label = `v-promo-${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: label })
  const cvId = (res.body as { id: string }).id

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
  const name1 = `Assoc1_${Date.now()}`
  const name2 = `Assoc2_${Date.now()}`
  const res1 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name1 })
  const res2 = await apiCall('POST', '/api/meta/v1/entity-types', { name: name2 })
  const etId1 = (res1.body as { entity_type: { id: string } }).entity_type.id
  const etId2 = (res2.body as { entity_type: { id: string } }).entity_type.id

  // Create association via API (PatternFly Select dropdowns are unreliable in E2E)
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/associations`, {
    target_entity_type_id: etId2, type: 'directional',
  })

  // Navigate and verify it shows in the UI
  await navigateToEntityType(name1)
  await pg.getByRole('tab', { name: /Associations/i }).click()
  await visible(pg.getByText(name2))
  await visible(pg.getByText('directional'))

  // Remove via UI
  await pg.getByRole('button', { name: 'Remove' }).click()

  // Verify removal via API
  const assocs = await apiCall('GET', `/api/meta/v1/entity-types/${etId1}/associations`)
  const items = (assocs.body as { items: { target_entity_type_id: string }[] }).items
  expect(items.length).toBe(0)

  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId1}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId2}`)
})

test('version history shows versions and diff', async () => {
  const name = `VerTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/entity-types', { name })
  const etId = (res.body as { entity_type: { id: string } }).entity_type.id

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

  // Clean up
  await apiCall('DELETE', `/api/meta/v1/enums/${found!.id}`)
})

test('navigate to enum detail and manage values', async () => {
  // Create enum with values via API
  const enumName = `ValTest_${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/enums', { name: enumName, values: ['alpha', 'beta'] })
  const enumId = (res.body as { id: string }).id

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

  await navigateToUI()
  await pg.getByRole('tab', { name: /Enums/i }).click()
  // Wait for enum list to load then refresh to ensure our new enum appears
  await pg.waitForTimeout(1000)
  await pg.getByRole('button', { name: 'Refresh' }).click()
  await visible(pg.getByText(enumName))

  // Click Delete
  await pg.getByRole('button', { name: 'Delete' }).first().click()
  await visible(pg.getByText('Confirm Deletion'))

  // Confirm
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  // Verify via API
  const check = await apiCall('GET', `/api/meta/v1/enums/${enumId}`)
  expect(check.status).toBe(404)
})

// ── Catalog Version Delete ──

test('delete catalog version via UI', async () => {
  const label = `v-del-${Date.now()}`
  const res = await apiCall('POST', '/api/meta/v1/catalog-versions', { version_label: label })
  const cvId = (res.body as { id: string }).id

  await navigateToUI()
  await pg.getByRole('tab', { name: /Catalog Versions/i }).click()
  await visible(pg.getByText(label))

  // Click Delete on the catalog version
  await pg.getByRole('button', { name: 'Delete' }).first().click()
  await visible(pg.getByText('Confirm Deletion'))
  await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

  // Verify deleted via API
  const check = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId}`)
  expect(check.status).toBe(404)
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

  // Add attributes to entity type 1 via API
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/attributes`, { name: 'hostname', type: 'string' })
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/attributes`, { name: 'memory_gb', type: 'number' })

  // Add association via API
  await apiCall('POST', `/api/meta/v1/entity-types/${etId1}/associations`, {
    target_entity_type_id: etId2, type: 'directional',
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
  await apiCall('DELETE', `/api/meta/v1/catalog-versions/${cvId}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId1}`)
  await apiCall('DELETE', `/api/meta/v1/entity-types/${etId2}`)
})
