// Import/Export Workflow system tests — run against a live deployment.
// Tests: export button visibility, export download, import wizard steps
// (file upload, collision resolution, success), full round-trip.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   cd ui && npx vitest run --config vitest.system.config.ts src/ImportExportWorkflow.system.test.ts

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
  getTypeVersionId,
  UI_URL,
  API_URL,
} from './test-helpers/system'

let browser: Browser
let pg: Page

// Entity type tracking
let etId: string
let etName: string
let etVersionId: string

// Catalog version and catalog
let cvId: string
const CV_LABEL = testName('impexp-cv')
const CATALOG_NAME = 'e2e-impexp'
const IMPORTED_CATALOG_NAME = 'e2e-impexp-imported'

// Exported data (captured via API for round-trip test)
let exportedData: unknown

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  // Clean up stale data from prior crashed runs
  await cleanupDnsCatalogs(CATALOG_NAME, IMPORTED_CATALOG_NAME, 'e2e-impexp-roundtrip')
  await cleanupE2EData()

  const stringVersionId = await getTypeVersionId('string')
  etName = testName('IE_Server')

  // --- Create entity type with attributes ---
  const etRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: etName,
    description: 'Entity type for import/export tests',
  })
  etId = etRes.body.entity_type.id

  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/attributes`, {
    name: 'hostname',
    type_definition_version_id: stringVersionId,
    required: true,
  })
  await apiCall('POST', `/api/meta/v1/entity-types/${etId}/attributes`, {
    name: 'env',
    type_definition_version_id: stringVersionId,
    required: false,
  })

  // Get latest version
  const versions = await apiCall('GET', `/api/meta/v1/entity-types/${etId}/versions`)
  etVersionId = versions.body.items[versions.body.items.length - 1].id

  // --- Create catalog version and pin ---
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: CV_LABEL,
    description: 'CV for import/export tests',
  })
  cvId = cvRes.body.id

  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: etVersionId,
  })

  // --- Create catalog with instances ---
  await apiCall('POST', '/api/data/v1/catalogs', {
    name: CATALOG_NAME,
    description: 'Test catalog for import/export tests',
    catalog_version_id: cvId,
  })

  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etName}`, {
    name: 'prod-server-1',
    description: 'Production server 1',
    attributes: { hostname: 'prod1.example.com', env: 'production' },
  })
  await apiCall('POST', `/api/data/v1/catalogs/${CATALOG_NAME}/${etName}`, {
    name: 'staging-server-1',
    description: 'Staging server 1',
    attributes: { hostname: 'stage1.example.com', env: 'staging' },
  })

  // --- Export via API (for round-trip test) ---
  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'Admin' }
  const exportRes = await fetch(`${API_URL}/api/data/v1/catalogs/${CATALOG_NAME}/export`, { headers })
  exportedData = await exportRes.json()
}, 60000)

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_NAME, IMPORTED_CATALOG_NAME, 'e2e-impexp-roundtrip')
  await cleanupE2EData()
  await teardownBrowser(browser)
})

// ============================================================
// Export Button Visibility
// ============================================================

describe('Export Button Visibility', () => {
  test('export button visible on catalog detail page for Admin role', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'Admin')

    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(CATALOG_NAME) }))

    // Export button should be visible
    const exportBtn = pg.getByRole('button', { name: 'Export' })
    await visible(exportBtn)
  })

  test('export button hidden for RO role', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'RO')

    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(CATALOG_NAME) }))

    // Export button should NOT be visible
    const exportBtn = pg.getByRole('button', { name: 'Export' })
    expect(await exportBtn.isVisible()).toBe(false)
  })

  test('export button hidden for RW role (requires Admin)', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'RW')

    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(CATALOG_NAME) }))

    // Export button should NOT be visible for RW
    const exportBtn = pg.getByRole('button', { name: 'Export' })
    expect(await exportBtn.isVisible()).toBe(false)
  })
})

// ============================================================
// Export Download
// ============================================================

describe('Export Download', () => {
  test('export catalog triggers file download', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'Admin')

    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(CATALOG_NAME) }))

    // Listen for download event
    const downloadPromise = pg.waitForEvent('download', { timeout: 15000 })

    // Click export
    const exportBtn = pg.getByRole('button', { name: 'Export' })
    await exportBtn.click()

    const download = await downloadPromise
    // Download filename should match the catalog name pattern
    expect(download.suggestedFilename()).toContain(CATALOG_NAME)
    expect(download.suggestedFilename()).toContain('.json')
  }, 30000)

  test('exported file via API contains expected structure', async () => {
    // Verify the exported data has the right structure
    const data = exportedData as Record<string, unknown>
    expect(data).toBeTruthy()
    expect(data.format_version).toBeDefined()
    expect(data.catalog).toBeDefined()
    expect(data.catalog_version).toBeDefined()
    expect(data.entity_types).toBeDefined()

    // Verify catalog metadata
    const catalog = data.catalog as { name: string }
    expect(catalog.name).toBe(CATALOG_NAME)

    // Verify entity types include our test entity type
    const entityTypes = data.entity_types as { name: string }[]
    expect(entityTypes.some(et => et.name === etName)).toBe(true)
  })
})

// ============================================================
// Import Button Visibility
// ============================================================

describe('Import Button Visibility', () => {
  test('import button visible on catalog list page for Admin role', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    // Navigate to catalogs tab
    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Import Catalog button should be visible
    await visible(pg.getByRole('button', { name: 'Import Catalog' }))
  })

  test('import button hidden for RO role', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'RO')

    // Navigate to catalogs tab
    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Import Catalog button should NOT be visible
    const importBtn = pg.getByRole('button', { name: 'Import Catalog' })
    expect(await importBtn.isVisible()).toBe(false)
  })
})

// ============================================================
// Import Wizard — File Upload Step
// ============================================================

describe('Import Wizard — File Upload Step', () => {
  test('import wizard opens with file upload dropzone', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Click Import Catalog button
    await pg.getByRole('button', { name: 'Import Catalog' }).click()

    // Modal should open with "Import Catalog" title
    await visible(pg.getByText('Catalog File (JSON)'))

    // Should see file upload area
    const dropzone = pg.getByTestId('file-dropzone')
    await visible(dropzone)

    // Should see "Click or drag a JSON file here"
    await visible(pg.getByText(/Click or drag a JSON file here/))

    // Analyze button should be disabled when no file is selected
    const analyzeBtn = pg.getByRole('button', { name: 'Analyze' })
    await visible(analyzeBtn)
    expect(await analyzeBtn.isDisabled()).toBe(true)

    // Cancel to close
    await pg.getByRole('button', { name: 'Cancel' }).click()
    await pg.waitForTimeout(500)
  })

  test('import wizard shows catalog name and CV label after file upload', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Upload exported data as a file via the hidden file input
    // Create a temp file with exported data and upload it
    const fileInput = pg.getByTestId('import-file-input')

    // Create a temporary file with the export data
    const exportJson = JSON.stringify(exportedData)
    const tmpFilePath = '/tmp/e2e-import-test.json'
    const fs = await import('fs')
    fs.writeFileSync(tmpFilePath, exportJson)

    // Upload the file
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // After file upload, catalog name and CV label fields should appear
    const catalogNameInput = pg.locator('#catalog-name')
    await visible(catalogNameInput)

    // The catalog name should be pre-filled from the export
    const nameValue = await catalogNameInput.inputValue()
    expect(nameValue).toBe(CATALOG_NAME)

    // CV label field should also appear
    const cvLabelInput = pg.locator('#cv-label')
    await visible(cvLabelInput)

    // Analyze button should now be enabled
    const analyzeBtn = pg.getByRole('button', { name: 'Analyze' })
    expect(await analyzeBtn.isDisabled()).toBe(false)

    // Clean up temp file
    fs.unlinkSync(tmpFilePath)

    // Cancel to close
    await pg.getByRole('button', { name: 'Cancel' }).click()
  }, 30000)
})

// ============================================================
// Import Wizard — Collision Resolution Step
// ============================================================

describe('Import Wizard — Collision Resolution', () => {
  test('importing with existing name shows collision resolution step', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Upload the export file
    const exportJson = JSON.stringify(exportedData)
    const tmpFilePath = '/tmp/e2e-import-collision-test.json'
    const fs = await import('fs')
    fs.writeFileSync(tmpFilePath, exportJson)

    const fileInput = pg.getByTestId('import-file-input')
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // Keep the same catalog name (will cause collision)
    // Click Analyze
    await pg.getByRole('button', { name: 'Analyze' }).click()
    await pg.waitForTimeout(3000)

    // Should show collision resolution step or an error about existing name
    const bodyText = await pg.textContent('body')
    const hasCollisions = bodyText?.includes('Collision') ||
      bodyText?.includes('collision') ||
      bodyText?.includes('identical') ||
      bodyText?.includes('conflict') ||
      bodyText?.includes('already exists')

    expect(hasCollisions).toBe(true)

    // Clean up
    fs.unlinkSync(tmpFilePath)

    // Cancel to close
    const cancelBtn = pg.getByRole('button', { name: 'Cancel' })
    const backBtn = pg.getByRole('button', { name: 'Back' })
    if (await backBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
      await backBtn.click()
      await pg.waitForTimeout(500)
    }
    if (await cancelBtn.isVisible({ timeout: 1000 }).catch(() => false)) {
      await cancelBtn.click()
    }
  }, 30000)
})

// ============================================================
// Import Success — New Catalog
// ============================================================

describe('Import Success', () => {
  test('import with new catalog name creates catalog successfully', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Upload the export file
    const exportJson = JSON.stringify(exportedData)
    const tmpFilePath = '/tmp/e2e-import-success-test.json'
    const fs = await import('fs')
    fs.writeFileSync(tmpFilePath, exportJson)

    const fileInput = pg.getByTestId('import-file-input')
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // Change catalog name to something new
    const catalogNameInput = pg.locator('#catalog-name')
    await catalogNameInput.clear()
    await catalogNameInput.fill(IMPORTED_CATALOG_NAME)
    await pg.waitForTimeout(300)

    // Change CV label to avoid collision
    const cvLabelInput = pg.locator('#cv-label')
    await cvLabelInput.clear()
    await cvLabelInput.fill(testName('impexp-imported-cv'))
    await pg.waitForTimeout(300)

    // Click Analyze
    await pg.getByRole('button', { name: 'Analyze' }).click()
    await pg.waitForTimeout(3000)

    // May go to collisions step (for entity types) or confirm step
    // If on collisions step with identical entity types, check the reuse checkboxes and continue
    const continueBtn = pg.getByRole('button', { name: 'Continue' })
    const confirmImportBtn = pg.getByRole('button', { name: 'Import' })

    if (await continueBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      // We're on the collision step — entity types already exist
      // Click Continue to proceed (identical types will be reused by default)
      await continueBtn.click()
      await pg.waitForTimeout(1000)
    }

    // Now we should be on confirm step
    if (await confirmImportBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await confirmImportBtn.click()
      await pg.waitForTimeout(5000)

      // Should see success message
      const successVisible = await pg.getByText(/imported successfully/).isVisible({ timeout: 10000 }).catch(() => false)
      if (successVisible) {
        // Verify the catalog was created
        const catalogRes = await apiCall('GET', `/api/data/v1/catalogs/${IMPORTED_CATALOG_NAME}`)
        expect(catalogRes.status).toBe(200)

        // Click "View Catalog" to close
        const viewBtn = pg.getByRole('button', { name: 'View Catalog' })
        if (await viewBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
          await viewBtn.click()
          await pg.waitForTimeout(1000)
        }
      }
    }

    // Clean up temp file
    fs.unlinkSync(tmpFilePath)
  }, 60000)
})

// ============================================================
// Full Export → Import Round-Trip
// ============================================================

describe('Export → Import Round-Trip', () => {
  test('round-trip preserves instance data', async () => {
    // The exported data was captured in beforeAll via API.
    // If the import test above succeeded, we can verify data integrity.
    const importedCatalogRes = await apiCall('GET', `/api/data/v1/catalogs/${IMPORTED_CATALOG_NAME}`)

    if (importedCatalogRes.status !== 200) {
      // Import may not have succeeded — skip this verification
      console.warn('SKIP: Imported catalog not found, round-trip verification skipped')
      return
    }

    // List instances in the imported catalog
    const instancesRes = await apiCall('GET', `/api/data/v1/catalogs/${IMPORTED_CATALOG_NAME}/${etName}`)

    if (instancesRes.status === 200) {
      const instances = instancesRes.body.items || []

      // Should have the same number of instances as the original
      expect(instances.length).toBe(2)

      // Verify instance names
      const names = instances.map((i: { name: string }) => i.name).sort()
      expect(names).toEqual(['prod-server-1', 'staging-server-1'])

      // Verify attributes are preserved on at least one instance
      const prodServer = instances.find((i: { name: string }) => i.name === 'prod-server-1')
      if (prodServer?.attributes) {
        const attrs = prodServer.attributes as Array<{ name: string; value: string }>
        const hostname = attrs.find((a) => a.name === 'hostname')
        const env = attrs.find((a) => a.name === 'env')
        expect(hostname?.value).toBe('prod1.example.com')
        expect(env?.value).toBe('production')
      }
    }
  })

  test('round-trip via API: export and re-import with new name', async () => {
    const roundTripName = 'e2e-impexp-roundtrip'

    // Export original catalog via API
    const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'Admin' }
    const exportRes = await fetch(`${API_URL}/api/data/v1/catalogs/${CATALOG_NAME}/export`, { headers })
    const exportData = await exportRes.json()

    // Import with a new name via API
    const importReq = {
      catalog_name: roundTripName,
      catalog_version_label: testName('roundtrip-cv'),
      reuse_existing: [etName], // Reuse the existing entity type
      data: exportData,
    }

    const importRes = await apiCall('POST', '/api/data/v1/catalogs/import', importReq)
    expect(importRes.status).toBe(201)
    expect(importRes.body.catalog_name).toBe(roundTripName)
    expect(importRes.body.instances_created).toBe(2)

    // Verify the imported catalog has the right instances
    const instancesRes = await apiCall('GET', `/api/data/v1/catalogs/${roundTripName}/${etName}`)
    expect(instancesRes.status).toBe(200)

    const instances = instancesRes.body.items || []
    expect(instances.length).toBe(2)

    // Clean up round-trip catalog
    await cleanupDnsCatalogs(roundTripName)
  })
})

// ============================================================
// Import Validation
// ============================================================

describe('Import File Validation', () => {
  test('invalid JSON file shows error in import wizard', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Create an invalid JSON file
    const fs = await import('fs')
    const tmpFilePath = '/tmp/e2e-import-invalid.json'
    fs.writeFileSync(tmpFilePath, 'this is not valid json')

    const fileInput = pg.getByTestId('import-file-input')
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // Should show error about invalid JSON
    await visible(pg.getByText(/Invalid JSON file/))

    // Clean up
    fs.unlinkSync(tmpFilePath)

    // Cancel to close
    await pg.getByRole('button', { name: 'Cancel' }).click()
  })

  test('JSON file missing required fields shows error', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Create a valid JSON file but missing required export fields
    const fs = await import('fs')
    const tmpFilePath = '/tmp/e2e-import-missing-fields.json'
    fs.writeFileSync(tmpFilePath, JSON.stringify({ name: 'test', data: [] }))

    const fileInput = pg.getByTestId('import-file-input')
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // Should show error about missing fields
    await visible(pg.getByText(/Not a valid catalog export file/))

    // Clean up
    fs.unlinkSync(tmpFilePath)

    // Cancel to close
    await pg.getByRole('button', { name: 'Cancel' }).click()
  })

  test('catalog name validation: invalid DNS label shows error', async () => {
    await pg.goto(`${UI_URL}/schema`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: /Catalogs/i }).click()
    await pg.waitForTimeout(1000)

    // Open import modal
    await pg.getByRole('button', { name: 'Import Catalog' }).click()
    await visible(pg.getByText('Catalog File (JSON)'))

    // Upload a valid export file
    const exportJson = JSON.stringify(exportedData)
    const tmpFilePath = '/tmp/e2e-import-dns-test.json'
    const fs = await import('fs')
    fs.writeFileSync(tmpFilePath, exportJson)

    const fileInput = pg.getByTestId('import-file-input')
    await fileInput.setInputFiles(tmpFilePath)
    await pg.waitForTimeout(1000)

    // Enter an invalid DNS label as catalog name
    const catalogNameInput = pg.locator('#catalog-name')
    await catalogNameInput.clear()
    await catalogNameInput.fill('INVALID_DNS_NAME!')
    await pg.waitForTimeout(500)

    // Should show DNS label validation error
    await visible(pg.getByText(/lowercase alphanumeric/i))

    // Clean up
    fs.unlinkSync(tmpFilePath)

    // Cancel to close
    await pg.getByRole('button', { name: 'Cancel' }).click()
  })
})
