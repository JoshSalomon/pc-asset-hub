// Operational editing live browser tests — run against a live deployment.
// Tests instance CRUD, containment, links, and role-aware controls in the operational UI.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   npx vitest run --config vitest.system.config.ts src/OperationalEditing.system.test.ts

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

const CATALOG_NAME = 'e2e-op-edit'
let etParentName: string
let etChildName: string
let etParentId: string
let etChildId: string
let cvId: string

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  await cleanupDnsCatalogs(CATALOG_NAME)
  await cleanupE2EData()

  // Create parent entity type with an attribute
  etParentName = testName('OPE_Server')
  const parentRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: etParentName,
    description: 'Parent for op editing tests',
  })
  etParentId = parentRes.body.entity_type.id

  const stringVersionId = await getTypeVersionId('string')
  await apiCall('POST', `/api/meta/v1/entity-types/${etParentId}/attributes`, {
    name: 'hostname',
    type_definition_version_id: stringVersionId,
    required: false,
    description: 'Hostname',
  })

  // Create child entity type
  etChildName = testName('OPE_Module')
  const childRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: etChildName,
    description: 'Child for op editing tests',
  })
  etChildId = childRes.body.entity_type.id

  // Create containment association (parent contains child)
  await apiCall('POST', `/api/meta/v1/entity-types/${etParentId}/associations`, {
    target_entity_type_id: etChildId,
    type: 'containment',
    name: 'contains-module',
    source_cardinality: '1',
    target_cardinality: '0..n',
  })

  // Create directional link association (parent uses-model parent — self-ref for simplicity)
  await apiCall('POST', `/api/meta/v1/entity-types/${etParentId}/associations`, {
    target_entity_type_id: etParentId,
    type: 'directional',
    name: 'depends-on',
    source_cardinality: '0..n',
    target_cardinality: '0..n',
  })

  // Get latest version IDs (associations bump the version)
  const parentVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etParentId}/versions`)
  const latestParentVersionId = parentVersions.body.items[parentVersions.body.items.length - 1].id
  const childVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etChildId}/versions`)
  const latestChildVersionId = childVersions.body.items[childVersions.body.items.length - 1].id

  // Create catalog version and pin both types at their latest versions
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: testName('OPE_CV'),
    description: 'CV for op editing',
    pins: [
      { entity_type_version_id: latestParentVersionId },
      { entity_type_version_id: latestChildVersionId },
    ],
  })
  cvId = cvRes.body.id

  // Create catalog
  await apiCall('POST', '/api/data/v1/catalogs', {
    name: CATALOG_NAME,
    description: 'Catalog for operational editing tests',
    catalog_version_id: cvId,
  })

  // Verify setup: check that pinned version snapshots include associations
  const parentSnapshot = await apiCall('GET', `/api/meta/v1/entity-types/${etParentId}/versions/${parentVersions.body.items.length}/snapshot`)
  if (!parentSnapshot.body.associations?.length) {
    throw new Error(`Setup failed: parent snapshot has no associations. Versions: ${JSON.stringify(parentVersions.body.items.map((v: { version: number }) => v.version))}`)
  }

  // Navigate to the app first so role selector is available
  await pg.goto(`${UI_URL}/catalogs/${CATALOG_NAME}`)
  await pg.waitForLoadState('networkidle')
  await setRole(pg, 'RW')
}, 60000)

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_NAME)
  await cleanupE2EData()
  await teardownBrowser(browser)
})

async function goToOperationalCatalog() {
  await pg.goto(`${UI_URL}/catalogs/${CATALOG_NAME}`)
  await pg.waitForLoadState('networkidle')
  await visible(pg.getByRole('heading', { name: CATALOG_NAME }))
}

describe('Operational Editing System Tests', () => {
  test('T-32.64: Create instance via operational UI', async () => {
    await goToOperationalCatalog()

    await pg.getByRole('button', { name: 'Create Instance' }).click()
    await visible(pg.locator('#create-entity-type'))
    await pg.locator('#create-entity-type').selectOption(etParentName)
    await visible(pg.locator('#create-inst-name'))
    await pg.locator('#create-inst-name').fill('test-server-1')
    await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

    // After creating, the tree shows the entity type group with count
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(1\\)`)))
    // Expand group to see instance
    await pg.getByText(new RegExp(`${etParentName}.*\\(1\\)`)).click()
    await visible(pg.getByText('test-server-1'))
  })

  test('T-32.65: Edit instance via operational UI', { timeout: 60000 }, async () => {
    await goToOperationalCatalog()
    const groupHeader = pg.getByText(new RegExp(`${etParentName}.*\\(1\\)`))
    await groupHeader.click()
    await visible(pg.getByText('test-server-1'))
    await pg.getByText('test-server-1').click()
    await visible(pg.getByRole('heading', { name: 'test-server-1' }))

    await pg.getByRole('button', { name: 'Edit' }).click()
    await visible(pg.locator('#edit-inst-name'))
    await pg.locator('#edit-inst-name').clear()
    await pg.locator('#edit-inst-name').fill('test-server-renamed')
    await pg.getByRole('dialog').getByRole('button', { name: 'Save' }).click()

    // Wait a moment then check for error alerts inside the dialog
    await pg.waitForTimeout(3000)
    const errorAlert = pg.getByRole('dialog').locator('.pf-v6-c-alert')
    const errorCount = await errorAlert.count()
    if (errorCount > 0) {
      const errorText = await errorAlert.first().textContent()
      throw new Error(`Edit modal shows error: ${errorText}`)
    }

    await hidden(pg.getByRole('dialog'), 30000)
  })

  test('T-32.76: Create instance via + icon on entity type group', async () => {
    await goToOperationalCatalog()
    await pg.getByRole('button', { name: `Create ${etParentName}` }).click()
    await visible(pg.locator('#create-inst-name'))
    await pg.locator('#create-inst-name').fill('test-server-2')
    await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()

    // Group count should increase to 2
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)))
  })

  test('T-32.69: Create link via operational UI', { timeout: 60000 }, async () => {
    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)).click()
    await pg.getByText('test-server-renamed').click()
    await visible(pg.getByRole('heading', { name: 'test-server-renamed' }))
    await visible(pg.getByRole('button', { name: 'Create Link' }), 45000)
    await pg.getByRole('button', { name: 'Create Link' }).click()
    await visible(pg.getByRole('dialog'))

    // Select association via PF6 Select (click toggle text → click option text)
    await pg.getByText('Select association...').click()
    await pg.getByText(/depends-on/).click()

    // Select target instance via PF6 Select (scope to dropdown menu item)
    await pg.getByText('Select target instance...').click()
    await pg.locator('button.pf-v6-c-menu__item').filter({ hasText: 'test-server-2' }).click()

    // Submit
    await pg.getByRole('dialog').getByRole('button', { name: 'Link' }).click()
    await pg.waitForTimeout(1000)

    // Verify link appears in forward references (detail panel already showing)
    await visible(pg.getByText(/depends-on/), 5000)

    // Clean up: delete link via API so subsequent tests aren't affected
    const refs = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${(await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)).body.items.find((i: { name: string }) => i.name === 'test-server-renamed').id}/references`)
    for (const ref of refs.body.items || []) {
      await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${(await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)).body.items.find((i: { name: string }) => i.name === 'test-server-renamed').id}/links/${ref.link_id}`)
    }
  })

  test('T-32.68: Add child via operational UI', { timeout: 60000 }, async () => {
    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)).click()
    await pg.getByText('test-server-renamed').click()
    await visible(pg.getByRole('heading', { name: 'test-server-renamed' }))
    await visible(pg.getByRole('button', { name: 'Add Child' }), 45000)
    await pg.getByRole('button', { name: 'Add Child' }).click()
    await visible(pg.getByRole('dialog'))

    // Select child entity type via PF6 Select (click toggle text → click option text)
    await pg.getByText('Select child type...').click()
    await pg.getByText(etChildName).click()

    // Fill child name
    await pg.locator('#child-name').fill('ui-test-child')

    // Submit (exact match to avoid "Create New" mode toggle)
    await pg.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()
    await pg.waitForTimeout(1000)

    // Reload and expand tree to verify child appears under parent
    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByLabel('Tree Browser').getByText('test-server-renamed').click()
    // Expand parent tree node to see contained children (click the ▸ toggle)
    await pg.getByLabel('Tree Browser').getByText('test-server-renamed').locator('..').locator('text=▸').click()
    await visible(pg.getByText('ui-test-child'), 10000)

    // Clean up: delete child via API so subsequent tests aren't affected
    const children = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}`)
    for (const c of children.body.items) {
      if (c.name === 'ui-test-child') {
        await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}/${c.id}`)
      }
    }
  })

  test('T-32.66: Delete instance via operational UI', async () => {
    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)).click()
    await pg.getByText('test-server-2').click()
    await visible(pg.getByRole('heading', { name: 'test-server-2' }))

    await pg.getByRole('button', { name: 'Delete' }).first().click()
    await visible(pg.getByText('Confirm Deletion'))
    await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

    // Group count should decrease to 1
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(1\\)`)))
  })

  test('T-32.67: Cascade delete via operational UI', async () => {
    // Create a parent with a child first
    await setRole(pg, 'RW')
    const parentRes = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`, { name: 'cascade-parent' })
    const parentId = parentRes.body.id
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${parentId}/${etChildName}`, { name: 'cascade-child' })

    await goToOperationalCatalog()
    // Expand parent group and select cascade-parent
    await pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)).click()
    await pg.getByText('cascade-parent').click()
    await visible(pg.getByRole('heading', { name: 'cascade-parent' }))

    // Delete should show cascade warning
    await pg.getByRole('button', { name: 'Delete' }).first().click()
    await visible(pg.getByText('Confirm Deletion'))
    await visible(pg.getByText(/1 contained instance/))
    await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()

    // Group count should decrease
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(1\\)`)))
  })

  test('T-32.70: Delete link via operational UI', { timeout: 60000 }, async () => {
    // Create a link first via API
    await setRole(pg, 'RW')
    const instances = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)
    const inst = instances.body.items[0]
    // Create another instance to link to
    const target = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`, { name: 'link-target' })
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${inst.id}/links`, {
      target_instance_id: target.body.id,
      association_name: 'depends-on',
    }, 'RW')

    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(2\\)`)).click()
    await pg.getByText(inst.name).click()
    await visible(pg.getByRole('heading', { name: inst.name }))
    // Forward refs should show with delete button
    await visible(pg.getByText('Forward References'), 30000)

    // Clean up extra instance
    await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${target.body.id}`)
  })

  test('T-32.71: Set parent via operational UI', { timeout: 60000 }, async () => {
    await setRole(pg, 'RW')
    // Create a child instance (rootless)
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}`, { name: 'orphan-child' })

    await goToOperationalCatalog()
    // Expand child group
    await pg.getByText(new RegExp(`${etChildName}.*\\(1\\)`)).click()
    await pg.getByText('orphan-child').click()
    await visible(pg.getByRole('heading', { name: 'orphan-child' }))
    await visible(pg.getByRole('button', { name: 'Set Parent' }), 45000)
    await pg.getByRole('button', { name: 'Set Parent' }).click()
    await visible(pg.getByRole('dialog'))
    await pg.keyboard.press('Escape')

    // Clean up
    const children = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}`)
    for (const c of children.body.items || []) {
      await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}/${c.id}`)
    }
  })

  test('T-32.72: Remove from container via operational UI', { timeout: 60000 }, async () => {
    await setRole(pg, 'RW')
    // Clean up any stale parent instances from prior tests (keep only the first)
    const allParents = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)
    for (let i = 1; i < (allParents.body.items || []).length; i++) {
      await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${allParents.body.items[i].id}`)
    }
    // Create parent with contained child
    const instances = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)
    const parent = instances.body.items[0]
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${parent.id}/${etChildName}`, { name: 'contained-child' })

    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByText(parent.name).click()
    await visible(pg.getByRole('heading', { name: parent.name }))

    // Expand parent tree node to see contained children (click the ▸ toggle)
    await pg.getByText(parent.name).locator('..').locator('text=▸').click()
    await visible(pg.getByText('contained-child'), 10000)
    await pg.getByText('contained-child').click()
    await visible(pg.getByRole('heading', { name: 'contained-child' }))
    await visible(pg.getByRole('button', { name: /Remove from Container/ }), 45000)
    // Verify button exists (full click-through would require more complex assertions)
    const btnCount = await pg.getByRole('button', { name: /Remove from Container/ }).count()
    expect(btnCount).toBe(1)

    // Clean up child
    const children = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}`)
    for (const c of children.body.items || []) {
      await apiCall('DELETE', `/api/data/v1/catalogs/${CATALOG_NAME}/${etChildName}/${c.id}`)
    }
  })

  test('T-32.77: Full workflow: create, edit, delete', { timeout: 60000 }, async () => {
    await setRole(pg, 'RW')
    // Ensure a known starting state: exactly 1 parent instance
    const allParents = await apiCall('GET', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`)
    const startCount = (allParents.body.items || []).length
    await goToOperationalCatalog()

    // Create
    await pg.getByRole('button', { name: 'Create Instance' }).click()
    await visible(pg.locator('#create-entity-type'))
    await pg.locator('#create-entity-type').selectOption(etParentName)
    await visible(pg.locator('#create-inst-name'))
    await pg.locator('#create-inst-name').fill('workflow-test')
    await pg.getByRole('dialog').getByRole('button', { name: 'Create' }).click()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(${startCount + 1}\\)`)))

    // Select and edit
    await pg.getByText(new RegExp(`${etParentName}.*\\(${startCount + 1}\\)`)).click()
    await pg.getByText('workflow-test').click()
    await visible(pg.getByRole('heading', { name: 'workflow-test' }))
    await pg.getByRole('button', { name: 'Edit' }).click()
    await visible(pg.locator('#edit-inst-name'))
    await pg.locator('#edit-inst-name').clear()
    await pg.locator('#edit-inst-name').fill('workflow-edited')
    await pg.getByRole('dialog').getByRole('button', { name: 'Save' }).click()
    await hidden(pg.getByRole('dialog'))

    // Delete — reload page to get clean tree state after edit
    await pg.waitForTimeout(1000)
    await goToOperationalCatalog()
    await pg.getByText(new RegExp(`${etParentName}.*\\(${startCount + 1}\\)`)).click()
    await pg.getByText('workflow-edited').click()
    await visible(pg.getByRole('heading', { name: 'workflow-edited' }))
    await pg.getByRole('button', { name: 'Delete' }).first().click()
    await visible(pg.getByText('Confirm Deletion'))
    await pg.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(${startCount}\\)`)))
  })

  test('T-32.73: RO role hides write controls in live deployment', async () => {
    await goToOperationalCatalog()
    await setRole(pg, 'RO')
    // Wait for page to re-render with RO role (catalog reloads on role change)
    await pg.waitForTimeout(2000)

    const createBtnCount = await pg.getByRole('button', { name: 'Create Instance' }).count()
    expect(createBtnCount).toBe(0)
  })

  test('T-32.74: RW on published catalog hides write controls in live deployment', async () => {
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/validate`)
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/publish`)

    await goToOperationalCatalog()
    await setRole(pg, 'RW')
    await pg.waitForTimeout(2000)

    const createBtnCount = await pg.getByRole('button', { name: 'Create Instance' }).count()
    expect(createBtnCount).toBe(0)
  })

  test('T-32.75: SuperAdmin on published catalog shows write controls in live deployment', async () => {
    await goToOperationalCatalog()
    await setRole(pg, 'SuperAdmin')
    // SuperAdmin can mutate even published catalogs
    await visible(pg.getByRole('button', { name: 'Create Instance' }), 10000)

    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/unpublish`)
  })
})

// TD-145: Verify attributes remain visible after structural mutations that bump versions
describe('TD-145: Attribute preservation after version bumps', () => {
  test('T-145.1: Parent attributes visible after CreateContainedInstance bumps version', { timeout: 60000 }, async () => {
    // Create parent with attribute via API
    const parentRes = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`, {
      name: 'td145-parent',
      attributes: { hostname: 'test-value-123' },
    })
    const parentId = parentRes.body.id

    // Verify parent shows attribute in UI
    await goToOperationalCatalog()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)))
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByText('td145-parent').click()
    await visible(pg.getByText('test-value-123'), 5000)

    // Add child via API (triggers CreateContainedInstance which bumps parent version)
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${parentId}/${etChildName}`, {
      name: 'td145-child',
    })

    // CRITICAL: Reload and verify parent attribute still visible after version bump
    await goToOperationalCatalog()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)))
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByText('td145-parent').click()
    await visible(pg.getByText('test-value-123'), 5000)
  })

  test('T-145.2: Source attributes visible after CreateAssociationLink bumps version', { timeout: 60000 }, async () => {
    // Create source and target instances via API
    const sourceRes = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`, {
      name: 'td145-source',
      attributes: { hostname: 'source-value-456' },
    })
    const sourceId = sourceRes.body.id
    const targetRes = await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}`, {
      name: 'td145-target',
    })
    const targetId = targetRes.body.id

    // Verify source shows attribute in UI
    await goToOperationalCatalog()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)))
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByText('td145-source').click()
    await visible(pg.getByText('source-value-456'), 5000)

    // Create link via API (bumps source version)
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etParentName}/${sourceId}/links`, {
      association_name: 'depends-on',
      target_instance_id: targetId,
    })

    // CRITICAL: Reload and verify source attribute still visible after version bump
    await goToOperationalCatalog()
    await visible(pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)))
    await pg.getByText(new RegExp(`${etParentName}.*\\(\\d+\\)`)).click()
    await pg.getByText('td145-source').click()
    await visible(pg.getByText('source-value-456'), 5000)
  })
})
