// Security flow system tests — run against a live deployment.
// Verifies role enforcement and stage guards across pages.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   npx vitest run --config vitest.system.config.ts src/SecurityFlows.system.test.ts

import { test, expect, beforeAll, afterAll, describe } from 'vitest'
import type { Browser, Page } from 'playwright'
import {
  setupBrowser,
  teardownBrowser,
  visible,
  setRole,
  apiCall,
  testName,
  cleanupE2EData,
  cleanupDnsCatalogs,
  UI_URL,
} from './test-helpers/system'

let browser: Browser
let pg: Page

// Test data IDs
let etId: string
let etvId: string
let cvId: string
let prodCvId: string
let catalogName: string

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  // Clean up stale data from prior crashed runs
  await cleanupDnsCatalogs('e2e-security')
  await cleanupE2EData()

  // Create entity type (returns both entity_type and version)
  const et = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('Sec_ET'),
    description: 'Security test entity type',
  }, 'SuperAdmin')
  etId = et.body.entity_type.id
  etvId = et.body.version.id

  // Add one attribute
  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/attributes`, {
    name: 'test_attr',
    type: 'string',
    required: true,
  }, 'SuperAdmin')

  // Create catalog version for the regular catalog
  const cv = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: testName('sec-cv'),
    description: 'Security test CV',
  }, 'SuperAdmin')
  cvId = cv.body.id

  // Get latest version (attribute creation creates new version)
  const versions = await apiCall('GET', `/api/meta/v1/entity-types/${etId}/versions`, undefined, 'SuperAdmin')
  const latestVersionId = versions.body.items[versions.body.items.length - 1].id

  // Pin entity type to CV
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: latestVersionId,
  }, 'SuperAdmin')

  // Create catalog (DNS-label compatible name)
  catalogName = 'e2e-security'
  await apiCall('POST', '/api/data/v1/catalogs', {
    name: catalogName,
    description: 'Security test catalog',
    catalog_version_id: cvId,
  }, 'SuperAdmin')

  // Create an instance so the catalog has data
  await apiCall('POST', `/api/data/v1/catalogs/${catalogName}/${testName('Sec_ET')}`, {
    name: 'sec-test-instance',
    description: 'Security test instance',
    attributes: {
      test_attr: 'test value',
    },
  }, 'SuperAdmin')

  // Validate the catalog
  await apiCall('POST', `/api/data/v1/catalogs/${catalogName}/validate`, undefined, 'SuperAdmin')

  // Publish the catalog
  await apiCall('POST', `/api/data/v1/catalogs/${catalogName}/publish`, undefined, 'SuperAdmin')

  // Create and promote a production CV for production stage tests
  const prodCv = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: testName('sec-prod-cv'),
    description: 'Security test production CV',
  }, 'SuperAdmin')
  prodCvId = prodCv.body.id

  // Pin entity type to production CV
  await apiCall('POST', `/api/meta/v1/catalog-versions/${prodCvId}/pins`, {
    entity_type_version_id: latestVersionId,
  }, 'SuperAdmin')

  // Promote to testing
  await apiCall('POST', `/api/meta/v1/catalog-versions/${prodCvId}/promote`, undefined, 'SuperAdmin')

  // Promote to production
  await apiCall('POST', `/api/meta/v1/catalog-versions/${prodCvId}/promote`, undefined, 'SuperAdmin')
})

afterAll(async () => {
  // Unpublish the catalog
  try {
    await apiCall('POST', `/api/data/v1/catalogs/${catalogName}/unpublish`, undefined, 'SuperAdmin')
  } catch {
    /* ignore */
  }

  // Delete catalog
  try {
    await apiCall('DELETE', `/api/data/v1/catalogs/${catalogName}`, undefined, 'SuperAdmin')
  } catch {
    /* ignore */
  }

  // Clean up all test data
  await cleanupE2EData()

  await teardownBrowser(browser)
})

// ============================================================
// Test 1: RO user - no Create/Edit/Delete buttons on any page
// ============================================================

describe('Test 1: RO user restrictions', () => {
  test('RO user: no Create Entity Type button on schema page', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'RO')

    await visible(pg.getByRole('tab', { name: 'Entity Types' }))
    await pg.getByRole('tab', { name: 'Entity Types' }).click()

    // No Create Entity Type button
    const createBtn = pg.getByRole('button', { name: 'Create Entity Type' })
    expect(await createBtn.isVisible()).toBe(false)
  })

  test('RO user: no edit controls on CV detail page', async () => {
    await pg.goto(`${UI_URL}/schema/catalog-versions`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'RO')

    // Navigate to first CV
    const firstCVLink = pg.getByRole('row').nth(1).getByRole('button').first()
    await firstCVLink.click()
    await visible(pg.getByRole('tab', { name: 'Overview' }))

    // No edit buttons
    expect(await pg.getByRole('button', { name: 'Edit version label' }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)
  })

  test('RO user: no pin controls on CV BOM tab', async () => {
    await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
    await visible(pg.getByRole('grid', { name: 'Pinned entity types' }))

    // No Add Pin button
    expect(await pg.getByRole('button', { name: 'Add Pin' }).isVisible()).toBe(false)

    // No Remove buttons
    const removeBtn = pg.getByRole('button', { name: 'Remove' }).first()
    expect(await removeBtn.isVisible()).toBe(false)
  })

  test('RO user: no Create Catalog button on catalog list', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'RO')

    await visible(pg.getByRole('grid'))

    // No Create Catalog button
    expect(await pg.getByRole('button', { name: 'Create Catalog' }).isVisible()).toBe(false)
  })

  test('RO user: no delete buttons on catalog list', async () => {
    await visible(pg.getByRole('grid'))

    // No Delete buttons in table rows
    const deleteBtn = pg.getByRole('button', { name: 'Delete' }).first()
    expect(await deleteBtn.isVisible()).toBe(false)
  })

  test('RO user: no edit controls on catalog detail', async () => {
    // Navigate to the test catalog
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'RO')

    const catalogRow = pg.getByRole('row').filter({ hasText: catalogName })
    await catalogRow.first().getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // No edit/delete/validate buttons
    expect(await pg.getByRole('button', { name: 'Edit', exact: true }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Delete' }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Validate' }).isVisible()).toBe(false)
  })
})

// ============================================================
// Test 2: Published catalog - Admin vs SuperAdmin
// ============================================================

describe('Test 2: Published catalog restrictions', () => {
  test('Published catalog: Admin sees no edit controls', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')

    // Navigate to published catalog (find one with published label)
    await visible(pg.getByRole('grid'))
    const publishedRow = pg.getByRole('row').filter({ hasText: 'published' })
    const count = await publishedRow.count()
    if (count === 0) {
      expect.fail('Prerequisite failed: no published catalog available')
    }

    await publishedRow.first().getByRole('button').first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))

    // Wait for page to fully load
    await pg.waitForLoadState('networkidle')

    // No Edit description button (inline edit for description)
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)

    // No Validate button
    expect(await pg.getByRole('button', { name: 'Validate' }).isVisible()).toBe(false)

    // Unpublish button should be visible (Admin can unpublish)
    await visible(pg.getByRole('button', { name: 'Unpublish' }))

    // Check for write protection alert (might not show if page thinks user is SuperAdmin)
    // This alert only shows for non-Admin roles on published catalogs
    const hasAlert = await pg.getByText('This catalog is published').isVisible()
    // If there's no alert, the user is probably SuperAdmin - which means role didn't stick
    if (!hasAlert) {
      console.log('WARN: Alert not shown - role may not have applied correctly')
    }
  })

  test('Published catalog: SuperAdmin sees all controls', async () => {
    // Stay on same page, just switch role
    await setRole(pg, 'SuperAdmin')

    // SuperAdmin should see Edit description button
    await visible(pg.getByRole('button', { name: 'Edit description' }))

    // SuperAdmin should see Validate button
    await visible(pg.getByRole('button', { name: 'Validate' }))

    // SuperAdmin should see Unpublish button
    await visible(pg.getByRole('button', { name: 'Unpublish' }))

    // Alert should NOT be shown for SuperAdmin
    expect(await pg.getByText('This catalog is published').isVisible()).toBe(false)
  })
})

// ============================================================
// Test 3: Testing CV - Admin vs SuperAdmin
// ============================================================

describe('Test 3: Testing CV stage guards', () => {
  test('Testing CV: Admin sees no edit/pin controls', async () => {
    await pg.goto(`${UI_URL}/schema/catalog-versions`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'Admin')

    // Find a testing-stage CV
    const testingRow = pg.getByRole('row').filter({ hasText: 'testing' })
    const count = await testingRow.count()
    if (count === 0) {
      expect.fail('Prerequisite failed: no testing-stage CV available')
    }

    await testingRow.first().getByRole('button').first().click()
    await visible(pg.getByRole('tab', { name: 'Overview' }))

    // No edit buttons on Overview tab
    expect(await pg.getByRole('button', { name: 'Edit version label' }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)

    // Check BOM tab
    await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
    await visible(pg.getByRole('grid', { name: 'Pinned entity types' }))

    // No Add Pin button
    expect(await pg.getByRole('button', { name: 'Add Pin' }).isVisible()).toBe(false)

    // No Remove buttons
    const removeBtn = pg.getByRole('button', { name: 'Remove' }).first()
    expect(await removeBtn.isVisible()).toBe(false)
  })

  test('Testing CV: SuperAdmin sees all controls', async () => {
    // Switch to SuperAdmin on same page
    await setRole(pg, 'SuperAdmin')

    // SuperAdmin should see Add Pin on BOM tab
    await visible(pg.getByRole('button', { name: 'Add Pin' }))

    // SuperAdmin should see Remove buttons
    const removeBtn = pg.getByRole('button', { name: 'Remove' }).first()
    await visible(removeBtn)

    // Check Overview tab
    await pg.getByRole('tab', { name: 'Overview' }).click()
    await visible(pg.getByRole('button', { name: 'Edit version label' }))
    await visible(pg.getByRole('button', { name: 'Edit description' }))
  })
})

// ============================================================
// Test 4: Production CV - nobody sees edit/pin controls
// ============================================================

describe('Test 4: Production CV stage guards', () => {
  test('Production CV: SuperAdmin sees no edit controls', async () => {
    await pg.goto(`${UI_URL}/schema/catalog-versions`)
    await pg.waitForLoadState('networkidle')
    await setRole(pg, 'SuperAdmin')

    // Find production CV
    const productionRow = pg.getByRole('row').filter({ hasText: 'production' })
    const count = await productionRow.count()
    if (count === 0) {
      expect.fail('Prerequisite failed: no production-stage CV available')
    }

    await productionRow.first().getByRole('button').first().click()
    await visible(pg.getByRole('tab', { name: 'Overview' }))

    // No edit buttons even for SuperAdmin
    expect(await pg.getByRole('button', { name: 'Edit version label' }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)
  })

  test('Production CV: SuperAdmin sees no pin controls', async () => {
    await pg.getByRole('tab', { name: 'Bill of Materials' }).click()
    await visible(pg.getByRole('grid', { name: 'Pinned entity types' }))

    // No Add Pin button
    expect(await pg.getByRole('button', { name: 'Add Pin' }).isVisible()).toBe(false)

    // No Remove buttons
    const removeBtn = pg.getByRole('button', { name: 'Remove' }).first()
    expect(await removeBtn.isVisible()).toBe(false)
  })

  test('Production CV: Admin sees no controls either', async () => {
    await setRole(pg, 'Admin')

    // Still on BOM tab
    expect(await pg.getByRole('button', { name: 'Add Pin' }).isVisible()).toBe(false)
    const removeBtn = pg.getByRole('button', { name: 'Remove' }).first()
    expect(await removeBtn.isVisible()).toBe(false)

    // Check Overview tab
    await pg.getByRole('tab', { name: 'Overview' }).click()
    expect(await pg.getByRole('button', { name: 'Edit version label' }).isVisible()).toBe(false)
    expect(await pg.getByRole('button', { name: 'Edit description' }).isVisible()).toBe(false)
  })
})

// ============================================================
// Test 5: Write-protected routes via API
// ============================================================

describe('Test 5: Published catalog API write protection', () => {
  test('PUT on published catalog as RW returns 403', async () => {
    const res = await apiCall('PUT', `/api/data/v1/catalogs/${catalogName}`, {
      description: 'attempt to edit',
    }, 'RW')

    expect(res.status).toBe(403)
    expect(res.body.message).toContain('published')
  })

  test('PUT on published catalog as Admin returns 403', async () => {
    const res = await apiCall('PUT', `/api/data/v1/catalogs/${catalogName}`, {
      description: 'attempt to edit',
    }, 'Admin')

    expect(res.status).toBe(403)
    expect(res.body.message).toContain('published')
  })

  test('DELETE on published catalog as RW returns 403', async () => {
    const res = await apiCall('DELETE', `/api/data/v1/catalogs/${catalogName}`, undefined, 'RW')

    expect(res.status).toBe(403)
  })

  test('DELETE on published catalog as Admin returns 403', async () => {
    const res = await apiCall('DELETE', `/api/data/v1/catalogs/${catalogName}`, undefined, 'Admin')

    expect(res.status).toBe(403)
  })

  test('SuperAdmin can edit published catalog', async () => {
    const res = await apiCall('PUT', `/api/data/v1/catalogs/${catalogName}`, {
      description: 'SuperAdmin edit',
    }, 'SuperAdmin')

    expect(res.status).toBe(200)
  })
})

// ============================================================
// Test 6: Stage guard via API
// ============================================================

describe('Test 6: Production CV API stage guards', () => {
  test('AddPin on production CV returns 400', async () => {
    const res = await apiCall('POST', `/api/meta/v1/catalog-versions/${prodCvId}/pins`, {
      entity_type_version_id: etvId,
    }, 'SuperAdmin')

    expect(res.status).toBe(400)
    expect(res.body.message).toContain('production')
  })

  test('UpdateCatalogVersion on production CV returns 400', async () => {
    const res = await apiCall('PUT', `/api/meta/v1/catalog-versions/${prodCvId}`, {
      description: 'attempt to edit production CV',
    }, 'SuperAdmin')

    expect(res.status).toBe(400)
    expect(res.body.message).toContain('production')
  })

  test('DeletePin on production CV returns 400', async () => {
    // Get a pin ID from the production CV
    const pins = await apiCall('GET', `/api/meta/v1/catalog-versions/${prodCvId}/pins`, undefined, 'SuperAdmin')
    const pinId = pins.body.items?.[0]?.pin_id

    if (!pinId) {
      expect.fail('Prerequisite failed: no pins on production CV')
    }

    const res = await apiCall('DELETE', `/api/meta/v1/catalog-versions/${prodCvId}/pins/${pinId}`, undefined, 'SuperAdmin')

    // DELETE should return 400 for production CV (stage guard)
    expect(res.status).toBe(400)
    // Note: body is null for DELETE in apiCall helper, even on error
  })
})
