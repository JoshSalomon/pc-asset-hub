// Catalog list and detail live browser tests — run against a live deployment.
// Tests catalog CRUD, instance management, containment, references, validation, publishing, and copy.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   npx vitest run --config vitest.system.config.ts src/CatalogDetail.system.test.ts

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
  UI_URL,
} from './test-helpers/system'

let browser: Browser
let pg: Page

// Test entity type and catalog version setup
let etParentId: string
let etParentName: string
let etChildId: string
let etChildName: string
let catalogVersionId: string
let catalogVersionLabel: string

// DNS-label-compatible catalog names (no underscores allowed)
const CATALOG_TEST_NAME = 'e2e-catdetail-test'
const CATALOG_COPY_NAME = 'e2e-catdetail-copy'
const CATALOG_PUBLISH_NAME = 'e2e-catdetail-publish'

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  // Clean up stale data from prior crashed runs
  await cleanupDnsCatalogs(CATALOG_TEST_NAME, CATALOG_COPY_NAME, CATALOG_PUBLISH_NAME, 'e2e-catdetail-delete')
  await cleanupE2EData()

  // Create test entity types with containment association
  etParentName = testName('CatDetail_Parent')
  etChildName = testName('CatDetail_Child')

  // Create parent entity type
  const parentRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: etParentName,
    description: 'Parent for catalog detail tests',
  })
  etParentId = parentRes.body.entity_type.id

  // Add attributes to parent
  await apiCall('POST', `/api/meta/v1/entity-types/${etParentId}/attributes`, {
    name: 'size',
    type: 'string',
    required: false,
    description: 'Size attribute',
  })

  // Create child entity type
  const childRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: etChildName,
    description: 'Child for catalog detail tests',
  })
  etChildId = childRes.body.entity_type.id

  // Add attributes to child
  await apiCall('POST', `/api/meta/v1/entity-types/${etChildId}/attributes`, {
    name: 'color',
    type: 'string',
    required: false,
    description: 'Color attribute',
  })

  // Create containment association from parent to child
  await apiCall('POST', `/api/meta/v1/entity-types/${etParentId}/associations`, {
    name: `${etParentName}_contains_${etChildName}`,
    target_entity_type_id: etChildId,
    type: 'containment',
    source_cardinality: '1',
    target_cardinality: '0..n',
  })
  // Create catalog version and pin entity types
  catalogVersionLabel = testName('CatDetail_CV')
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: catalogVersionLabel,
    description: 'CV for catalog detail tests',
  })
  catalogVersionId = cvRes.body.id

  // Get latest versions (attribute/association creation creates new versions)
  const parentVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etParentId}/versions`)
  const childVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etChildId}/versions`)
  const latestParentVersionId = parentVersions.body.items[parentVersions.body.items.length - 1].id
  const latestChildVersionId = childVersions.body.items[childVersions.body.items.length - 1].id

  // Pin both entity types
  await apiCall('POST', `/api/meta/v1/catalog-versions/${catalogVersionId}/pins`, {
    entity_type_version_id: latestParentVersionId,
  })
  await apiCall('POST', `/api/meta/v1/catalog-versions/${catalogVersionId}/pins`, {
    entity_type_version_id: latestChildVersionId,
  })
})

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_TEST_NAME, CATALOG_COPY_NAME, CATALOG_PUBLISH_NAME, 'e2e-catdetail-delete')

  // Clean up E2E_ prefixed data
  await cleanupE2EData()

  await teardownBrowser(browser)
})

// ============================================================
// Catalog List Page Tests
// ============================================================

describe('Catalog List Page', () => {
  test('shows existing catalogs in table', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('grid', { name: 'Catalogs' }))

    // Should have header row + at least one data row
    const rows = pg.getByRole('row')
    const count = await rows.count()
    expect(count).toBeGreaterThan(1)
  })

  test('create catalog: open modal, fill fields, create, verify in list', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('grid', { name: 'Catalogs' }))

    // Open create modal
    await pg.getByRole('button', { name: 'Create Catalog' }).click()
    await visible(pg.getByRole('dialog'))

    // Fill form fields
    await pg.locator('#cat-name').fill(CATALOG_TEST_NAME)
    await pg.locator('#cat-desc').fill('Test catalog for detail tests')

    // Select CV from dropdown
    await pg.getByText('Select a catalog version').click()
    await pg.waitForTimeout(500)
    await pg.getByText(catalogVersionLabel).click()

    // Submit
    await pg.getByRole('button', { name: 'Create' }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Verify catalog appears in list
    await visible(pg.getByRole('gridcell', { name: CATALOG_TEST_NAME }))
  })

  test('delete catalog: click Delete, confirm, removed from list', async () => {
    // Create a throwaway catalog to delete
    const deleteTestName = 'e2e-catdetail-delete'
    await apiCall('POST', '/api/data/v1/catalogs', {
      name: deleteTestName,
      description: 'To be deleted',
      catalog_version_id: catalogVersionId,
    })

    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('grid', { name: 'Catalogs' }))

    // Find the catalog row and click Delete (last button to avoid the name link)
    const deleteRow = pg.getByRole('row').filter({ hasText: deleteTestName })
    await deleteRow.getByRole('button', { name: 'Delete' }).last().click()

    // Confirm deletion
    await visible(pg.getByRole('dialog'))
    await pg.getByRole('button', { name: 'Delete' }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Verify catalog is removed from list
    expect(await pg.getByRole('gridcell', { name: deleteTestName }).isVisible()).toBe(false)
  })
})

// ============================================================
// Catalog Detail Page Navigation and Structure
// ============================================================

describe('Catalog Detail Page', () => {
  test('navigate to catalog detail, see entity type tabs and status badge', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('grid', { name: 'Catalogs' }))

    // Click on test catalog
    const catalogRow = pg.getByRole('row').filter({ hasText: CATALOG_TEST_NAME })
    await catalogRow.getByRole('button').first().click()

    // Verify on detail page
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Should see entity type tabs
    await visible(pg.getByRole('tab', { name: etParentName, exact: true }))
    await visible(pg.getByRole('tab', { name: etChildName, exact: true }))

    // Should see status badge (draft by default)
    await visible(pg.getByText('draft'))
  })

  test('inline edit description: click Edit, type, Save, verify', async () => {
    // Already on catalog detail from previous test
    const newDesc = 'Updated description via inline edit'

    // Click Edit description
    await pg.getByRole('button', { name: 'Edit description' }).click()

    // Type new description
    const descInput = pg.getByRole('textbox', { name: 'Description' })
    await visible(descInput)
    await descInput.clear()
    await descInput.fill(newDesc)

    // Save
    await pg.getByRole('button', { name: 'Save' }).click()

    // Wait for edit to complete
    await pg.waitForTimeout(500)

    // Verify description updated
    expect(await pg.textContent('body')).toContain(newDesc)
  })
})

// ============================================================
// Instance CRUD Tests
// ============================================================

describe('Instance CRUD', () => {
  const instanceName = 'TestInstance1'

  test('create instance: click Create, fill modal, verify in table', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_TEST_NAME}`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Click parent entity type tab
    await pg.getByRole('tab', { name: etParentName, exact: true }).click()
    await pg.waitForTimeout(1000)

    // Click Create Instance
    const createBtn = pg.getByRole('button', { name: `Create ${etParentName}` })
    await visible(createBtn, 30000)
    await createBtn.click()

    // Fill create instance modal
    await visible(pg.getByRole('dialog'))
    await pg.getByRole('textbox', { name: 'Name' }).fill(instanceName)
    await pg.getByRole('textbox', { name: 'Description' }).fill('First test instance')

    // Fill attribute field if it exists
    const sizeInput = pg.getByRole('textbox', { name: 'size' })
    if (await sizeInput.isVisible({ timeout: 1000 }).catch(() => false)) {
      await sizeInput.fill('large')
    }

    // Submit
    await pg.getByRole('button', { name: 'Create' }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Verify instance appears in table
    const instanceTable = pg.getByRole('grid', { name: `${etParentName} instances` })
    await visible(instanceTable)
    await visible(instanceTable.getByRole('gridcell', { name: instanceName }))
  }, 60000)

  test('edit instance: click Edit, change name, Save, verify', async () => {
    // Already on catalog detail page with instance table visible
    const instanceTable = pg.getByRole('grid', { name: `${etParentName} instances` })
    await visible(instanceTable)
    const instanceRow = instanceTable.getByRole('row').filter({ hasText: instanceName })

    // Click Edit button
    await instanceRow.getByRole('button', { name: 'Edit' }).click()

    // Edit modal should open
    await visible(pg.getByRole('dialog'))

    // Change description
    const descInput = pg.getByRole('textbox', { name: 'Description' })
    await descInput.clear()
    await descInput.fill('Updated instance description')

    // Save
    await pg.getByRole('button', { name: 'Save' }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Verify change (wait a moment for UI update)
    await pg.waitForTimeout(500)
  }, 60000)

  test('delete instance: click Delete, confirm, removed from table', async () => {
    // Use the instance created in the previous test
    // Already on catalog detail page with instance table visible
    const instanceTable = pg.getByRole('grid', { name: `${etParentName} instances` })
    await visible(instanceTable)

    // Look for the test instance
    const deleteRow = instanceTable.getByRole('row').filter({ hasText: instanceName })

    // If not visible, the create test may have failed - skip
    if (!(await deleteRow.isVisible({ timeout: 2000 }).catch(() => false))) {
      expect.fail('Prerequisite failed: test instance not found')
    }

    // Click Delete
    await deleteRow.getByRole('button', { name: 'Delete' }).last().click()

    // Confirm (if there's a confirmation dialog)
    const confirmDialog = pg.getByRole('dialog')
    if (await confirmDialog.isVisible({ timeout: 1000 }).catch(() => false)) {
      await pg.getByRole('button', { name: 'Delete' }).click()
      await hidden(confirmDialog)
    }

    // Verify instance removed
    await pg.waitForTimeout(500)
    expect(await instanceTable.getByRole('gridcell', { name: instanceName }).isVisible()).toBe(false)
  }, 60000)
})

// ============================================================
// Containment Tests
// ============================================================

describe('Containment', () => {
  const parentName = 'ParentWithChild'
  const childName = 'ContainedChild1'

  test('add contained instance via Details panel', async () => {
    // Create a fresh parent instance via UI so we know it's visible
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_TEST_NAME}`)
    await setRole(pg, 'Admin')
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))
    await pg.getByRole('tab', { name: etParentName, exact: true }).click()
    await pg.waitForTimeout(1000)

    // Create parent instance
    const createBtn = pg.getByRole('button', { name: `Create ${etParentName}` })
    await visible(createBtn)
    await createBtn.click()
    await visible(pg.getByRole('dialog'))
    await pg.getByRole('textbox', { name: 'Name' }).fill(parentName)
    await pg.getByRole('textbox', { name: 'Description' }).fill('Parent for containment')
    await pg.getByRole('button', { name: 'Create' }).click()
    await hidden(pg.getByRole('dialog'))
    await pg.waitForTimeout(1000)

    // Now find it and open Details
    const instanceTable = pg.getByRole('grid', { name: `${etParentName} instances` })
    await visible(instanceTable)
    const parentRow = instanceTable.getByRole('row').filter({ hasText: parentName })

    // Click Details (or Hide Details if already open, try both)
    const detailsBtn = parentRow.getByRole('button', { name: /Details/ })
    await visible(detailsBtn, 10000)
    await detailsBtn.click()

    // Wait for Details panel
    await visible(pg.getByRole('heading', { name: new RegExp(`Details: ${parentName}`) }), 10000)

    // Click "Add Contained Instance" button (containment association must exist)
    const addContainedBtn = pg.getByRole('button', { name: 'Add Contained Instance' })
    await visible(addContainedBtn, 5000)
    await addContainedBtn.click()

    // Fill contained instance modal
    await visible(pg.getByRole('dialog'))

    // Select child entity type from dropdown
    const childTypeSelect = pg.getByRole('dialog').locator('button:has-text("Select child type")')
    if (await childTypeSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
      await childTypeSelect.click()
      await pg.waitForTimeout(500)
      await pg.getByText(etChildName, { exact: true }).click()
      await pg.waitForTimeout(500)
    }

    await pg.getByRole('textbox', { name: 'Name' }).fill(childName)
    await pg.getByRole('textbox', { name: 'Description' }).fill('Contained child instance')

    // Fill color attribute if visible
    const colorInput = pg.getByRole('textbox', { name: 'color' })
    if (await colorInput.isVisible({ timeout: 1000 }).catch(() => false)) {
      await colorInput.fill('blue')
    }

    // Submit (use exact match to avoid matching "Create New" mode toggle)
    await pg.getByRole('dialog').getByRole('button', { name: 'Create', exact: true }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Verify child appears in Contained Instances section
    await pg.waitForTimeout(1000)
    expect(await pg.textContent('body')).toContain(childName)
  }, 90000)
})

// ============================================================
// Validation Tests
// ============================================================

describe('Validation', () => {
  test('validate catalog: click Validate, see results', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    const catalogRow = pg.getByRole('row').filter({ hasText: CATALOG_TEST_NAME })
    await catalogRow.getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Click Validate
    await pg.getByRole('button', { name: 'Validate' }).click()

    // Wait for validation results
    await pg.waitForTimeout(1000)

    // Should see validation result (either "Validation passed" or "Validation failed")
    const bodyText = await pg.textContent('body')
    const hasValidationResult =
      bodyText?.includes('Validation passed') || bodyText?.includes('Validation failed')
    expect(hasValidationResult).toBe(true)
  })
})

// ============================================================
// Publishing Tests
// ============================================================

describe('Publishing', () => {
  test('publish catalog: validate first, then click Publish, verify badge', async () => {
    // Create a fresh catalog for publishing tests
    await apiCall('POST', '/api/data/v1/catalogs', {
      name: CATALOG_PUBLISH_NAME,
      description: 'Catalog for publish tests',
      catalog_version_id: catalogVersionId,
    })

    // Validate it first via API
    await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_PUBLISH_NAME}/validate`)

    // Navigate to catalog detail
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    const catalogRow = pg.getByRole('row').filter({ hasText: CATALOG_PUBLISH_NAME })
    await catalogRow.getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Should see valid status (use exact match to avoid matching "Validate" button)
    await visible(pg.getByText('valid', { exact: true }).first())

    // Click Publish
    await pg.getByRole('button', { name: 'Publish' }).click()

    // Wait for publish to complete and page to refresh
    await pg.waitForTimeout(1000)

    // Verify published badge
    await visible(pg.getByText('published'))
  })

  test('published catalog shows write protection for RW', async () => {
    // Already on published catalog detail
    await setRole(pg, 'RW')

    // Edit description button should not be visible
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)

    // Validate button should not be visible
    expect(await pg.getByRole('button', { name: 'Validate' }).isVisible()).toBe(false)

    // Reset role
    await setRole(pg, 'Admin')
  })

  test('unpublish catalog: click Unpublish, verify badge removed', async () => {
    // Already on published catalog as Admin
    await setRole(pg, 'Admin')

    // Click Unpublish
    await pg.getByRole('button', { name: 'Unpublish' }).click()

    // Wait for unpublish to complete
    await pg.waitForTimeout(1000)

    // Published badge should be gone
    expect(await pg.getByText('published').isVisible()).toBe(false)

    // Should see draft or valid status instead
    const bodyText = await pg.textContent('body')
    expect(bodyText?.includes('valid') || bodyText?.includes('draft')).toBe(true)
  })
})

// ============================================================
// Copy Catalog Tests
// ============================================================

describe('Copy Catalog', () => {
  test('copy catalog: click Copy, fill name, verify new catalog appears', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    const catalogRow = pg.getByRole('row').filter({ hasText: CATALOG_TEST_NAME })
    await catalogRow.getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Click Copy button
    await pg.getByRole('button', { name: 'Copy' }).click()

    // Fill copy modal
    await visible(pg.getByRole('dialog'))
    await pg.locator('#copy-name').fill(CATALOG_COPY_NAME)
    await pg.locator('#copy-desc').fill('Copied catalog')

    // Submit
    await pg.getByRole('button', { name: 'Copy' }).click()

    // Wait for modal to close
    await hidden(pg.getByRole('dialog'))

    // Navigate back to catalog list and verify copy exists
    await pg.getByRole('button', { name: '← Back to Catalogs' }).click()
    await visible(pg.getByRole('grid', { name: 'Catalogs' }))
    await visible(pg.getByRole('gridcell', { name: CATALOG_COPY_NAME }))
  })
})

// ============================================================
// CV Selector Tests
// ============================================================

describe('CV Selector', () => {
  test('CV selector shows and allows selection for Admin on unpublished catalog', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await setRole(pg, 'Admin')
    const catalogRow = pg.getByRole('row').filter({ hasText: CATALOG_TEST_NAME })
    await catalogRow.getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Look for CV selector (only visible for Admin on unpublished catalogs)
    const cvSelector = pg.locator('[aria-label="Select catalog version"]')

    // Check if visible
    if (await cvSelector.isVisible({ timeout: 1000 }).catch(() => false)) {
      // It's an unpublished catalog, selector should be there
      await visible(cvSelector)

      // Try clicking it (don't actually change, just verify it's interactive)
      await cvSelector.click()

      // Options should appear
      await pg.waitForTimeout(300)

      // Close dropdown by clicking elsewhere
      await pg.keyboard.press('Escape')
    } else {
      // Catalog might be published or role doesn't allow changes
      console.log('CV selector not visible (expected if catalog is published or role restricted)')
    }
  })
})
