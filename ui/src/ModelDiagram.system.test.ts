// Model Diagram system tests — run against a live deployment.
// Tests: diagram rendering on CV detail and catalog detail pages,
// entity type nodes, association edges, containment edges.
//
// Prerequisites:
//   - kind cluster running with deployed app
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   cd ui && npx vitest run --config vitest.system.config.ts src/ModelDiagram.system.test.ts

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
} from './test-helpers/system'

let browser: Browser
let pg: Page

// Entity type IDs and version IDs
let etServerId: string
let etServerVersionId: string
let etDiskId: string
let etDiskVersionId: string
let etNicId: string
let etNicVersionId: string

// Catalog version and catalog
let cvId: string
const CV_LABEL = testName('diagram-cv')
const CATALOG_NAME = 'e2e-diagram'

// Empty catalog for empty-state test
const EMPTY_CATALOG_NAME = 'e2e-diagram-empty'
let emptyCvId: string

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page

  // Clean up stale data from prior crashed runs
  await cleanupDnsCatalogs(CATALOG_NAME, EMPTY_CATALOG_NAME)
  await cleanupE2EData()

  const stringVersionId = await getTypeVersionId('string')

  // --- Create entity type: Server ---
  const serverRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('Diag_Server'),
    description: 'Server for diagram tests',
  })
  etServerId = serverRes.body.entity_type.id

  await apiCall('POST', `/api/meta/v1/entity-types/${etServerId}/attributes`, {
    name: 'hostname',
    type_definition_version_id: stringVersionId,
    required: true,
  })

  // --- Create entity type: Disk ---
  const diskRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('Diag_Disk'),
    description: 'Disk for diagram tests',
  })
  etDiskId = diskRes.body.entity_type.id

  await apiCall('POST', `/api/meta/v1/entity-types/${etDiskId}/attributes`, {
    name: 'capacity',
    type_definition_version_id: stringVersionId,
    required: false,
  })

  // --- Create entity type: NIC ---
  const nicRes = await apiCall('POST', '/api/meta/v1/entity-types', {
    name: testName('Diag_NIC'),
    description: 'NIC for diagram tests',
  })
  etNicId = nicRes.body.entity_type.id

  await apiCall('POST', `/api/meta/v1/entity-types/${etNicId}/attributes`, {
    name: 'mac_address',
    type_definition_version_id: stringVersionId,
    required: true,
  })

  // --- Create containment association: Server contains Disk ---
  await apiCall('POST', `/api/meta/v1/entity-types/${etServerId}/associations`, {
    name: `${testName('Diag_Server')}_contains_${testName('Diag_Disk')}`,
    target_entity_type_id: etDiskId,
    type: 'containment',
    source_cardinality: '1',
    target_cardinality: '0..n',
  })

  // --- Create reference association: Server references NIC ---
  await apiCall('POST', `/api/meta/v1/entity-types/${etServerId}/associations`, {
    name: `${testName('Diag_Server')}_refs_${testName('Diag_NIC')}`,
    target_entity_type_id: etNicId,
    type: 'directional',
    source_cardinality: '0..n',
    target_cardinality: '0..n',
  })

  // --- Get latest versions (attributes/associations create new versions) ---
  const serverVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etServerId}/versions`)
  const diskVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etDiskId}/versions`)
  const nicVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etNicId}/versions`)

  etServerVersionId = serverVersions.body.items[serverVersions.body.items.length - 1].id
  etDiskVersionId = diskVersions.body.items[diskVersions.body.items.length - 1].id
  etNicVersionId = nicVersions.body.items[nicVersions.body.items.length - 1].id

  // --- Create catalog version and pin entity types ---
  const cvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: CV_LABEL,
    description: 'CV for diagram tests',
  })
  cvId = cvRes.body.id

  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: etServerVersionId,
  })
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: etDiskVersionId,
  })
  await apiCall('POST', `/api/meta/v1/catalog-versions/${cvId}/pins`, {
    entity_type_version_id: etNicVersionId,
  })

  // --- Create catalog ---
  await apiCall('POST', '/api/data/v1/catalogs', {
    name: CATALOG_NAME,
    description: 'Test catalog for diagram tests',
    catalog_version_id: cvId,
  })

  // --- Create empty catalog version (no pins) for empty-state test ---
  const emptyCvRes = await apiCall('POST', '/api/meta/v1/catalog-versions', {
    version_label: testName('diagram-empty-cv'),
    description: 'Empty CV for diagram empty state test',
  })
  emptyCvId = emptyCvRes.body.id

  await apiCall('POST', '/api/data/v1/catalogs', {
    name: EMPTY_CATALOG_NAME,
    description: 'Empty catalog for diagram empty state test',
    catalog_version_id: emptyCvId,
  })
}, 60000)

afterAll(async () => {
  await cleanupDnsCatalogs(CATALOG_NAME, EMPTY_CATALOG_NAME)
  await cleanupE2EData()
  await teardownBrowser(browser)
})

// ============================================================
// Diagram on Catalog Version Detail Page
// ============================================================

describe('Diagram on Catalog Version Detail Page', () => {
  test('Diagram tab renders entity type nodes', async () => {
    await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
    await setRole(pg, 'Admin')

    // Click Diagram tab
    await pg.getByRole('tab', { name: 'Diagram' }).click()
    await pg.waitForTimeout(2000)

    // The diagram container should be visible
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram, 10000)

    // Verify entity type names appear in the SVG diagram
    // The node labels include "(V{n})" suffix
    const svgContent = await diagram.textContent()
    expect(svgContent).toContain(testName('Diag_Server'))
    expect(svgContent).toContain(testName('Diag_Disk'))
    expect(svgContent).toContain(testName('Diag_NIC'))
  }, 30000)

  test('Diagram shows attribute labels in entity type nodes', async () => {
    // Already on diagram tab from previous test
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram)

    const svgContent = await diagram.textContent()

    // Attributes should appear in the diagram nodes
    expect(svgContent).toContain('hostname')
    expect(svgContent).toContain('capacity')
    expect(svgContent).toContain('mac_address')
  })

  test('Diagram shows association edges between entity types', async () => {
    // Still on diagram tab
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram)

    const svgContent = await diagram.textContent()

    // Edge labels contain the association name and cardinalities
    const refAssocName = `${testName('Diag_Server')}_refs_${testName('Diag_NIC')}`
    expect(svgContent).toContain(refAssocName)
  })

  test('Diagram shows containment edges with diamond marker', async () => {
    // Still on diagram tab — containment edges use diamond-source marker
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram)

    // The containment edge renders a <g data-testid="diamond-source"> marker
    const diamondMarker = diagram.locator('[data-testid="diamond-source"]')
    expect(await diamondMarker.count()).toBeGreaterThanOrEqual(1)

    // Containment association label should be visible
    const svgContent = await diagram.textContent()
    const containsAssocName = `${testName('Diag_Server')}_contains_${testName('Diag_Disk')}`
    expect(svgContent).toContain(containsAssocName)
  })
})

// ============================================================
// Diagram on Catalog Detail Page (Schema)
// ============================================================

describe('Diagram on Catalog Detail Page', () => {
  test('Model Diagram tab renders on catalog detail page', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${CATALOG_NAME}`)
    await setRole(pg, 'Admin')

    // Wait for page to load
    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(CATALOG_NAME) }))

    // Click Model Diagram tab
    await pg.getByRole('tab', { name: 'Model Diagram' }).click()
    await pg.waitForTimeout(2000)

    // The diagram container should be visible
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram, 10000)

    // Verify entity type names appear
    const svgContent = await diagram.textContent()
    expect(svgContent).toContain(testName('Diag_Server'))
    expect(svgContent).toContain(testName('Diag_Disk'))
    expect(svgContent).toContain(testName('Diag_NIC'))
  }, 30000)

  test('Catalog diagram shows same edges as CV diagram', async () => {
    // Already on Model Diagram tab
    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram)

    const svgContent = await diagram.textContent()

    // Both containment and reference edges should be present
    const containsAssocName = `${testName('Diag_Server')}_contains_${testName('Diag_Disk')}`
    const refAssocName = `${testName('Diag_Server')}_refs_${testName('Diag_NIC')}`
    expect(svgContent).toContain(containsAssocName)
    expect(svgContent).toContain(refAssocName)
  })
})

// ============================================================
// Empty Diagram State
// ============================================================

describe('Diagram Empty State', () => {
  test('Empty diagram shows "No model diagram available" when no entity types pinned', async () => {
    await pg.goto(`${UI_URL}/schema/catalog-versions/${emptyCvId}`)
    await setRole(pg, 'Admin')

    // Click Diagram tab
    await pg.getByRole('tab', { name: 'Diagram' }).click()
    await pg.waitForTimeout(1500)

    // Should see empty state message
    await visible(pg.getByText(/No model diagram available/))
  })

  test('Empty catalog detail page shows "No entity types pinned" instead of diagram', async () => {
    await pg.goto(`${UI_URL}/schema/catalogs/${EMPTY_CATALOG_NAME}`)
    await setRole(pg, 'Admin')

    // Wait for page to load
    await visible(pg.getByRole('heading', { level: 2, name: new RegExp(EMPTY_CATALOG_NAME) }))

    // Empty catalog has no pins, so no tabs are rendered — just the empty state
    await visible(pg.getByText(/No entity types pinned/))
  })
})

// ============================================================
// Diagram Updates After Schema Change
// ============================================================

describe('Diagram Updates After Schema Change', () => {
  test('Diagram updates after adding a new association', async () => {
    // Create a new association: NIC references Disk
    await apiCall('POST', `/api/meta/v1/entity-types/${etNicId}/associations`, {
      name: `${testName('Diag_NIC')}_refs_${testName('Diag_Disk')}`,
      target_entity_type_id: etDiskId,
      type: 'directional',
      source_cardinality: '0..n',
      target_cardinality: '0..1',
    })

    // Get the new NIC version after association added
    const nicVersions = await apiCall('GET', `/api/meta/v1/entity-types/${etNicId}/versions`)
    const newNicVersionId = nicVersions.body.items[nicVersions.body.items.length - 1].id

    // Update the pin to point to the new NIC version
    const pinsRes = await apiCall('GET', `/api/meta/v1/catalog-versions/${cvId}/pins`)
    const nicPin = pinsRes.body.items.find(
      (p: { entity_type_name: string }) => p.entity_type_name === testName('Diag_NIC')
    )
    if (nicPin) {
      await apiCall('PUT', `/api/meta/v1/catalog-versions/${cvId}/pins/${nicPin.pin_id}`, {
        entity_type_version_id: newNicVersionId,
      })
    }

    // Navigate to diagram and verify the new edge appears
    await pg.goto(`${UI_URL}/schema/catalog-versions/${cvId}`)
    await setRole(pg, 'Admin')

    await pg.getByRole('tab', { name: 'Diagram' }).click()
    await pg.waitForTimeout(2000)

    const diagram = pg.getByTestId('entity-type-diagram')
    await visible(diagram, 10000)

    const svgContent = await diagram.textContent()
    const newAssocName = `${testName('Diag_NIC')}_refs_${testName('Diag_Disk')}`
    expect(svgContent).toContain(newAssocName)
  }, 30000)
})
