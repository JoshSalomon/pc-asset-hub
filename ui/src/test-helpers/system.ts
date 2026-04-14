// Shared helpers for live browser system tests.
// These tests run against a deployed instance (kind cluster).
//
// Prerequisites:
//   - kind cluster running with services exposed
//   - UI at http://localhost:30000, API at http://localhost:30080

import { chromium, type Browser, type Page, type Locator } from 'playwright'

// Base URLs for live services
export const UI_URL = process.env.UI_URL || 'http://localhost:30000'
export const API_URL = process.env.API_URL || 'http://localhost:30080'

// Setup browser for testing
export async function setupBrowser(): Promise<{ browser: Browser; page: Page }> {
  // Check API health before launching browser
  const health = await fetch(`${API_URL}/healthz`)
  if (!health.ok) {
    throw new Error(`API not reachable at ${API_URL}`)
  }

  const headless = process.env.HEADLESS !== 'false'
  const slowMo = process.env.SLOWMO ? Number(process.env.SLOWMO) : undefined
  const browser = await chromium.launch({ headless, slowMo })
  const page = await browser.newPage()
  return { browser, page }
}

// Teardown browser
export async function teardownBrowser(browser: Browser) {
  await browser?.close()
}

// Wait for element to be visible
export async function visible(locator: Locator, timeout = 15000) {
  await locator.waitFor({ state: 'visible', timeout })
}

// Wait for element to be hidden
export async function hidden(locator: Locator, timeout = 15000) {
  await locator.waitFor({ state: 'hidden', timeout })
}

// Set user role via the role selector in the masthead
export async function setRole(page: Page, role: 'RO' | 'RW' | 'Admin' | 'SuperAdmin') {
  // Click the role selector button (shows current role)
  await page.getByRole('button', { name: /Role:/i }).click()

  // Select the desired role from the dropdown (exact match to avoid Admin matching SuperAdmin)
  await page.getByRole('option', { name: role, exact: true }).click()

  // Wait for the role to be reflected in the button
  await visible(page.getByRole('button', { name: new RegExp(`Role: ${role}`, 'i') }))
}

// Make an API call with optional role header
export async function apiCall(
  method: string,
  path: string,
  body?: object,
  role: 'RO' | 'RW' | 'Admin' | 'SuperAdmin' = 'Admin'
): Promise<{ status: number; body: any }> {
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-User-Role': role,
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (method === 'DELETE' && res.ok) {
    return { status: res.status, body: null }
  }

  const text = await res.text()
  try {
    return { status: res.status, body: JSON.parse(text) }
  } catch {
    return { status: res.status, body: text }
  }
}

// Navigate to the schema page and wait for it to load
export async function navigateToSchema(page: Page) {
  await page.goto(`${UI_URL}/schema`)
  await page.waitForLoadState('networkidle')

  // Wait for the main heading and tabs to be visible
  await visible(page.getByText('AI Asset Hub'))
  await visible(page.getByRole('tab', { name: /Entity Types/i }))
  await visible(page.getByRole('tab', { name: /Catalog Versions/i }))
}

// Clean up specific DNS-labeled catalogs (not covered by cleanupE2EData)
export async function cleanupDnsCatalogs(...names: string[]) {
  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'SuperAdmin' }
  for (const name of names) {
    try {
      await fetch(`${API_URL}/api/data/v1/catalogs/${name}/unpublish`, { method: 'POST', headers })
    } catch { /* ignore */ }
    try {
      await fetch(`${API_URL}/api/data/v1/catalogs/${name}`, { method: 'DELETE', headers })
    } catch { /* ignore */ }
  }
}

// Generate test data name with E2E_ prefix
export function testName(base: string): string {
  return `E2E_${base}`
}

// Look up a system type definition's latest version ID by name
// Caches results so multiple calls don't hit the API repeatedly
const typeVersionCache: Record<string, string> = {}

export async function getTypeVersionId(typeName: string): Promise<string> {
  if (typeVersionCache[typeName]) return typeVersionCache[typeName]

  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'Admin' }
  const res = await (await fetch(`${API_URL}/api/meta/v1/type-definitions`, { headers })).json()
  const td = res.items?.find((t: { name: string }) => t.name === typeName)
  if (!td) throw new Error(`Type definition '${typeName}' not found`)

  const versions = await (await fetch(`${API_URL}/api/meta/v1/type-definitions/${td.id}/versions`, { headers })).json()
  const latest = versions.items?.[versions.items.length - 1]
  if (!latest) throw new Error(`No versions found for type definition '${typeName}'`)

  typeVersionCache[typeName] = latest.id
  return latest.id
}

// Clean up all test data with E2E_ prefix
export async function cleanupE2EData() {
  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'SuperAdmin' }

  // Clean catalogs first (they may have instances and reference CVs)
  try {
    const catalogs = await (await fetch(`${API_URL}/api/data/v1/catalogs`, { headers })).json()
    for (const catalog of catalogs.items || []) {
      if (catalog.name.startsWith('E2E_')) {
        // Unpublish if published
        if (catalog.published) {
          try {
            await fetch(`${API_URL}/api/data/v1/catalogs/${catalog.name}/unpublish`, {
              method: 'POST',
              headers,
            })
          } catch {
            /* ignore unpublish errors */
          }
        }
        // Delete catalog
        try {
          await fetch(`${API_URL}/api/data/v1/catalogs/${catalog.name}`, {
            method: 'DELETE',
            headers,
          })
        } catch {
          /* ignore delete errors */
        }
      }
    }
  } catch {
    /* ignore */
  }

  // Clean catalog versions
  try {
    const cvs = await (await fetch(`${API_URL}/api/meta/v1/catalog-versions`, { headers })).json()
    for (const cv of cvs.items || []) {
      if (cv.version_label.startsWith('E2E_')) {
        await fetch(`${API_URL}/api/meta/v1/catalog-versions/${cv.id}`, {
          method: 'DELETE',
          headers,
        })
      }
    }
  } catch {
    /* ignore */
  }

  // Clean entity types
  try {
    const ets = await (await fetch(`${API_URL}/api/meta/v1/entity-types`, { headers })).json()
    for (const et of ets.items || []) {
      if (et.name.startsWith('E2E_')) {
        await fetch(`${API_URL}/api/meta/v1/entity-types/${et.id}`, {
          method: 'DELETE',
          headers,
        })
      }
    }
  } catch {
    /* ignore */
  }

  // Clean type definitions (non-system only)
  try {
    const tds = await (await fetch(`${API_URL}/api/meta/v1/type-definitions`, { headers })).json()
    for (const td of tds.items || []) {
      if (!td.system && td.name.startsWith('E2E_')) {
        await fetch(`${API_URL}/api/meta/v1/type-definitions/${td.id}`, {
          method: 'DELETE',
          headers,
        })
      }
    }
  } catch {
    /* ignore */
  }
}
