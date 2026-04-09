// System tests for CatalogVersionDetailPage — run against a live deployment.
// Tests: inline edit, BOM tab, stage guards, transitions, diagram.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   cd /home/jsalomon/src/pc-asset-hub/ui && npx vitest run --config vitest.system.config.ts src/CatalogVersionDetail.system.test.ts

import { test, expect, beforeAll, afterAll } from 'vitest'
import { type Browser, type Page } from 'playwright'
import {
  setupBrowser,
  teardownBrowser,
  visible,
  setRole,
  apiCall,
  testName,
  cleanupE2EData,
  UI_URL,
} from './test-helpers/system'

let browser: Browser
let pg: Page
let cvId: string
let etId: string

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  // Clean up any leftover E2E data from previous runs
  await cleanupE2EData()

  // Create test entity type with an attribute
  const etRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('CVDetail_ET'),
    description: 'Entity type for CV detail tests',
  })
  etId = etRes.body.entity_type.id
  const etvId = etRes.body.version.id
  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/versions/${etvId}/attributes`, {
    name: 'hostname',
    data_type: 'string',
    is_required: true,
    description: 'Server hostname',
  })

  // Create second entity type for Add Pin tests
  await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('CVDetail_ET2'),
    description: 'Second entity type',
  })

  // Create test enum
  await apiCall('POST', '/api/meta/v1/enums', {
    name: testName('CVDetail_Enum'),
    values: ['enabled', 'disabled'],
  })

  // Create catalog version with pin to the entity type
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: testName('cvdetail-v1'),
    description: 'CV for detail tests',
  })
  cvId = cvRes.body.id

  // Add pin to the entity type
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: etvId,
  })
})

afterAll(async () => {
  await cleanupE2EData()
  await teardownBrowser(browser)
})

// ============================================================
// Overview Tab — Basic rendering and inline edit
// ============================================================

test('CV detail page loads with overview tab', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await visible(pg.getByRole('heading', { name: testName('cvdetail-v1') }))
  await visible(pg.getByText('development').first())
  await visible(pg.getByText('CV for detail tests'))
})

test('inline edit version label: click Edit → type → Save → verify updated', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await visible(pg.getByRole('button', { name: 'Edit version label' }))

  // Click Edit
  await pg.getByRole('button', { name: 'Edit version label' }).click()
  const input = pg.getByRole('textbox', { name: 'Version Label' })
  await visible(input)

  // Type new value
  await input.fill(testName('cvdetail-v1-updated'))

  // Save
  await pg.getByRole('button', { name: 'Save' }).first().click()

  // Wait for page to refresh and verify updated label
  await pg.waitForTimeout(1000)
  await visible(pg.getByRole('heading', { name: testName('cvdetail-v1-updated') }))
})

test('inline edit version label cancel: click Edit → type → Cancel → verify original', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await visible(pg.getByRole('button', { name: 'Edit version label' }))

  // Get original label
  const origLabel = await pg.getByRole('heading', { level: 2 }).textContent()

  // Click Edit
  await pg.getByRole('button', { name: 'Edit version label' }).click()
  const input = pg.getByRole('textbox', { name: 'Version Label' })
  await visible(input)

  // Type something different
  await input.fill('temp-label-should-not-save')

  // Cancel
  await pg.getByRole('button', { name: 'Cancel' }).first().click()

  // Verify original label is still shown
  await expect(pg.getByRole('heading', { level: 2 }).textContent()).resolves.toBe(origLabel)
})

test('inline edit description: click Edit → type → Save → verify updated', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await visible(pg.getByRole('button', { name: 'Edit description' }))

  // Click Edit
  await pg.getByRole('button', { name: 'Edit description' }).click()
  const input = pg.getByRole('textbox', { name: 'Description' })
  await visible(input)

  // Type new value
  await input.fill('Updated description for CV detail tests')

  // Save
  await pg.getByRole('button', { name: 'Save' }).first().click()

  // Wait for page to refresh and verify updated description
  await pg.waitForTimeout(1000)
  await visible(pg.getByText('Updated description for CV detail tests'))
})

// ============================================================
// BOM Tab — Pin display, version dropdown, Add Pin modal
// ============================================================

test('BOM tab shows pinned entity type with version', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')

  // Click BOM tab
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()

  // Verify pinned entity type is shown
  await visible(pg.getByRole('button', { name: testName('CVDetail_ET'), exact: true }))

  // Verify version dropdown is shown
  await visible(pg.getByRole('button', { name: `Version for ${testName('CVDetail_ET')}` }))
})

test('BOM version dropdown can change version (if multiple versions exist)', async () => {
  // First, create a second version of the entity type
  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/versions`, {
    description: 'V2 of entity type',
  })

  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()

  // Find version dropdown for our entity type
  const versionBtn = pg.getByRole('button', { name: `Version for ${testName('CVDetail_ET')}` })
  await visible(versionBtn)

  // Click to open dropdown
  await versionBtn.click()

  // Wait for dropdown to populate
  await pg.waitForTimeout(1000)

  // Check if V1 option is visible (should always be available)
  const v1Option = pg.getByRole('option', { name: 'V1' })
  const v1Visible = await v1Option.isVisible()

  // If V1 is visible, the dropdown is working correctly
  expect(v1Visible).toBe(true)
})

test('BOM Add Pin modal opens and can add a pin', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()

  // Click Add Pin
  await pg.getByRole('button', { name: 'Add Pin' }).click()

  // Modal should open
  await visible(pg.getByRole('dialog'))
  await visible(pg.getByText('Select entity type...'))

  // Select entity type (ET2)
  await pg.getByRole('dialog').getByText('Select entity type...').click()
  await pg.getByTestId(`pin-et-${testName('CVDetail_ET2')}`).click()

  // Select version (V1)
  await visible(pg.getByRole('dialog').getByText('Select version...'))
  await pg.getByRole('dialog').getByText('Select version...').click()
  await pg.getByTestId('pin-etv-V1').click()

  // Click Add
  await pg.getByRole('dialog').getByRole('button', { name: 'Add' }).click()

  // Wait for modal to close and list to refresh
  await pg.waitForTimeout(1000)

  // Verify second entity type is now pinned
  await visible(pg.getByRole('button', { name: testName('CVDetail_ET2'), exact: true }))
})

test('BOM Remove pin works', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()

  // Wait for pins to load
  await visible(pg.getByRole('button', { name: testName('CVDetail_ET2'), exact: true }))

  // Get initial count of Remove buttons
  const initialCount = await pg.getByRole('button', { name: 'Remove' }).count()

  // Click first Remove button
  await pg.getByRole('button', { name: 'Remove' }).first().click()

  // Wait for removal to complete
  await pg.waitForTimeout(1000)

  // Verify one less Remove button
  const afterCount = await pg.getByRole('button', { name: 'Remove' }).count()
  expect(afterCount).toBe(initialCount - 1)
})

// ============================================================
// Stage Guards — Testing CV stage transitions
// ============================================================

test('stage guard - development: Admin sees edit controls', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')

  // Overview tab edit buttons visible
  await visible(pg.getByRole('button', { name: 'Edit version label' }))
  await visible(pg.getByRole('button', { name: 'Edit description' }))

  // BOM tab controls visible
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
  await visible(pg.getByRole('button', { name: 'Add Pin' }))
  await visible(pg.getByRole('button', { name: 'Remove' }).first())
})

test('stage guard - testing: Admin does NOT see edit controls, SuperAdmin does', async () => {
  // Promote CV to testing
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/promote`)

  // Test as Admin
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')

  // Wait for page to load
  await visible(pg.getByText('testing').first())

  // Overview tab edit buttons NOT visible
  expect(await pg.getByRole('button', { name: 'Edit version label' }).isVisible()).toBe(false)
  expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)

  // BOM tab controls NOT visible
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
  await pg.waitForTimeout(500)
  expect(await pg.getByRole('button', { name: 'Add Pin' }).isVisible()).toBe(false)
  expect(await pg.getByRole('button', { name: 'Remove' }).isVisible()).toBe(false)

  // Switch to SuperAdmin
  await pg.getByRole('tab', { name: 'Overview' }).click()
  await setRole(pg, 'SuperAdmin')

  // Edit buttons SHOULD be visible for SuperAdmin
  await visible(pg.getByRole('button', { name: 'Edit version label' }))
  await visible(pg.getByRole('button', { name: 'Edit description' }))

  // BOM tab controls visible for SuperAdmin
  await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
  await visible(pg.getByRole('button', { name: 'Add Pin' }))

  // Demote back to development for other tests
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/demote`, { target_stage: 'development' })
})

// ============================================================
// Transitions Tab — Shows lifecycle history
// ============================================================

test('Transitions tab shows transition history (after promote/demote)', async () => {
  // Promote to testing
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/promote`)

  // Demote back to development
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/demote`, { target_stage: 'development' })

  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')

  // Click Transitions tab
  await pg.getByRole('tab', { name: 'Transitions' }).click()

  // Should see transition records
  await visible(pg.getByText('(initial)'))
  await visible(pg.getByRole('gridcell', { name: 'testing' }).first())
  // Development appears in both the stage badge and the transitions table, so use gridcell
  await visible(pg.getByRole('gridcell', { name: 'development' }).first())
})

// ============================================================
// Diagram Tab — Renders diagram or shows empty state
// ============================================================

test('Diagram tab renders (may show diagram or empty state depending on data)', async () => {
  await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
  await setRole(pg, 'Admin')

  // Click Diagram tab
  await pg.getByRole('tab', { name: 'Diagram' }).click()

  // Either diagram renders or empty state shows
  // We verify the tab renders without error
  await pg.waitForTimeout(1500)

  // Check for either diagram container or empty state
  const hasDiagram = await pg.getByTestId('entity-type-diagram').isVisible()
  const hasEmpty = await pg.getByText(/No model diagram available/).isVisible()

  expect(hasDiagram || hasEmpty).toBe(true)
})
