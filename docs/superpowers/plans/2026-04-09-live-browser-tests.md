# Live Browser Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add comprehensive Playwright-based browser tests that run against the live deployed UI (Kind cluster), covering all pages and features not yet covered by the existing 30 system tests.

**Architecture:** Tests use Playwright directly (not Vitest Browser Mode) to launch a headless browser and interact with the live deployment. Each test file covers one page/feature area. Tests create their own data via the API, run UI interactions, then clean up. A shared test helpers module provides common patterns (navigation, role switching, API calls, cleanup). Tests are excluded from `test-all` since they require a running cluster — invoked explicitly via `make test-system` or a new `make test-e2e` target.

**Tech Stack:** Vitest + Playwright (direct, not browser mode), live API at `localhost:30080`, live UI at `localhost:30000`

---

## Current State

### Existing system tests (`App.system.test.ts` — 30 tests)
Covers: entity type CRUD, filtering, deletion (correct row), attributes, associations, enums, catalog versions (create/promote/delete), role switching (RO hides controls), full workflow, cardinality.

### NOT covered by system tests
- **Catalog detail page** — instance CRUD, containment, links, references, validation, publish/unpublish
- **Catalog version detail page** — inline edit (label, description), BOM tab (pin add/remove/update), stage guards, transitions, diagram
- **Catalog list page** — create catalog, delete, search
- **Operational data viewer** — tree browser, instance detail panel, containment navigation, reference navigation
- **Security flows** — published catalog write protection, stage guard enforcement across roles
- **Landing page** — navigation to schema management and catalogs

### Constraints
- Tests run ONLY on the host machine with Kind cluster deployed (not from isolated container)
- Chrome/Chromium must be installed (`npx playwright install chromium`)
- Tests must not be part of `test-all` — they need explicit invocation
- Tests must clean up all data they create (prefix-based cleanup pattern)
- Tests should be runnable individually (`npx vitest run --config vitest.system.config.ts -t "test name"`)

---

## File Structure

```
ui/
  src/
    test-helpers/
      system.ts                              # Shared helpers: navigation, roles, API, cleanup
    App.system.test.ts                       # EXISTING — 30 entity type + enum + CV tests
    CatalogList.system.test.ts               # NEW — catalog list page (create, delete, search)
    CatalogDetail.system.test.ts             # NEW — catalog detail (instances, refs, validation, publish)
    CatalogVersionDetail.system.test.ts      # NEW — CV detail (inline edit, BOM, stage guards)
    DataViewer.system.test.ts                # NEW — operational data viewer (tree, detail, navigation)
    SecurityFlows.system.test.ts             # NEW — cross-cutting security (role enforcement, stage guards)
    LandingPage.system.test.ts              # NEW — landing page navigation
  vitest.system.config.ts                    # EXISTING — includes src/**/*.system.test.ts
Makefile                                     # MODIFY — add test-e2e target
```

---

## Task 1: Extract Shared Test Helpers

**Files:**
- Create: `ui/src/test-helpers/system.ts`
- Modify: `ui/src/App.system.test.ts` — import helpers instead of inline definitions

The existing `App.system.test.ts` has inline helpers (`navigateToUI`, `apiCall`, `visible`, `hidden`, `cleanupTestData`, `setRole`, `trackResource`). Extract them to a shared module so all new test files can use them.

- [ ] **Step 1: Create shared helpers module**

```typescript
// ui/src/test-helpers/system.ts
import { expect } from 'vitest'
import { chromium, type Browser, type Page, type Locator } from 'playwright'

export const UI_URL = process.env.UI_URL || 'http://localhost:30000'
export const API_URL = process.env.API_URL || 'http://localhost:30080'

let browser: Browser
let page: Page

export async function setupBrowser(): Promise<{ browser: Browser; page: Page }> {
  const health = await fetch(`${API_URL}/healthz`)
  if (!health.ok) throw new Error('API not reachable at ' + API_URL)
  browser = await chromium.launch({ headless: true })
  page = await browser.newPage()
  return { browser, page }
}

export async function teardownBrowser() {
  await browser?.close()
}

export function getPage(): Page {
  return page
}

export async function visible(locator: Locator, timeout = 15000) {
  await locator.waitFor({ state: 'visible', timeout })
}

export async function hidden(locator: Locator, timeout = 15000) {
  await locator.waitFor({ state: 'hidden', timeout })
}

export async function setRole(pg: Page, role: 'RO' | 'RW' | 'Admin' | 'SuperAdmin') {
  const roleBtn = pg.locator('button').filter({ hasText: /Role:/ }).first()
  await roleBtn.click()
  await pg.getByRole('option', { name: role }).click()
  await pg.waitForTimeout(300)
}

export async function apiCall(
  method: string, path: string, body?: object, role = 'Admin'
): Promise<{ status: number; body: any }> {
  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-User-Role': role },
    body: body ? JSON.stringify(body) : undefined,
  })
  if (method === 'DELETE') return { status: res.status, body: null }
  const text = await res.text()
  try { return { status: res.status, body: JSON.parse(text) } }
  catch { return { status: res.status, body: text } }
}

export async function navigateToSchema(pg: Page) {
  await pg.goto(`${UI_URL}/schema`)
  await visible(pg.getByRole('tablist'))
}

export async function navigateToTab(pg: Page, tabName: string) {
  await pg.getByRole('tab', { name: tabName }).click()
  await pg.waitForTimeout(300)
}

// Prefix-based cleanup for test data
const TEST_PREFIX = 'E2E_'

export function testName(base: string): string {
  return `${TEST_PREFIX}${base}`
}

export async function cleanupE2EData() {
  const headers = { 'Content-Type': 'application/json', 'X-User-Role': 'SuperAdmin' }

  // Clean catalogs first (depend on CVs)
  try {
    const cats = await (await fetch(`${API_URL}/api/data/v1/catalogs`, { headers })).json()
    for (const cat of cats.items || []) {
      if (cat.name.startsWith(TEST_PREFIX)) {
        if (cat.published) {
          await fetch(`${API_URL}/api/data/v1/catalogs/${cat.name}/unpublish`, { method: 'POST', headers })
        }
        await fetch(`${API_URL}/api/data/v1/catalogs/${cat.name}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }

  // Clean CVs
  try {
    const cvs = await (await fetch(`${API_URL}/api/meta/v1/catalog-versions`, { headers })).json()
    for (const cv of cvs.items || []) {
      if (cv.version_label.startsWith(TEST_PREFIX)) {
        await fetch(`${API_URL}/api/meta/v1/catalog-versions/${cv.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }

  // Clean entity types
  try {
    const ets = await (await fetch(`${API_URL}/api/meta/v1/entity-types`, { headers })).json()
    for (const et of ets.items || []) {
      if (et.name.startsWith(TEST_PREFIX)) {
        await fetch(`${API_URL}/api/meta/v1/entity-types/${et.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }

  // Clean enums
  try {
    const enums = await (await fetch(`${API_URL}/api/meta/v1/enums`, { headers })).json()
    for (const en of enums.items || []) {
      if (en.name.startsWith(TEST_PREFIX)) {
        await fetch(`${API_URL}/api/meta/v1/enums/${en.id}`, { method: 'DELETE', headers })
      }
    }
  } catch { /* ignore */ }
}
```

- [ ] **Step 2: Verify the module compiles**

Run: `cd ui && npx tsc --noEmit`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add ui/src/test-helpers/system.ts
git commit -m "Add shared test helpers for live browser tests"
```

---

## Task 2: Landing Page Tests

**Files:**
- Create: `ui/src/LandingPage.system.test.ts`

- [ ] **Step 1: Write landing page tests**

```typescript
// ui/src/LandingPage.system.test.ts
import { test, expect, beforeAll, afterAll } from 'vitest'
import { type Browser, type Page } from 'playwright'
import { setupBrowser, teardownBrowser, UI_URL, visible } from './test-helpers/system'

let browser: Browser
let pg: Page

beforeAll(async () => {
  const setup = await setupBrowser()
  browser = setup.browser
  pg = setup.page
})

afterAll(async () => { await teardownBrowser() })

test('landing page loads with title and sections', async () => {
  await pg.goto(UI_URL)
  await visible(pg.getByRole('heading', { name: 'AI Asset Hub' }))
  await visible(pg.getByText('Schema Management'))
  await visible(pg.getByText('Catalogs'))
})

test('schema management card navigates to /schema', async () => {
  await pg.goto(UI_URL)
  await visible(pg.getByText('Entity Types & Model'))
  await pg.getByText('Entity Types & Model').click()
  await expect(pg).toHaveURL(/\/schema/)
  await visible(pg.getByRole('tab', { name: 'Entity Types' }))
})

test('catalog card navigates to catalog detail', async () => {
  await pg.goto(UI_URL)
  // Wait for catalogs to load
  await pg.waitForTimeout(1000)
  // If catalogs exist, clicking one should navigate
  const catalogCards = pg.locator('[class*="card"]').filter({ hasText: /draft|valid|invalid/ })
  const count = await catalogCards.count()
  if (count > 0) {
    await catalogCards.first().click()
    await visible(pg.getByRole('button', { name: '← Back to Catalogs' }))
  }
})

test('role selector shows all 4 roles', async () => {
  await pg.goto(UI_URL)
  const roleBtn = pg.locator('button').filter({ hasText: /Role:/ }).first()
  await roleBtn.click()
  await visible(pg.getByRole('option', { name: 'RO' }))
  await visible(pg.getByRole('option', { name: 'RW' }))
  await visible(pg.getByRole('option', { name: 'Admin' }))
  await visible(pg.getByRole('option', { name: 'SuperAdmin' }))
  // Close dropdown
  await pg.keyboard.press('Escape')
})
```

- [ ] **Step 2: Run tests**

Run: `cd ui && npx vitest run --config vitest.system.config.ts src/LandingPage.system.test.ts`
Expected: 4 tests pass

- [ ] **Step 3: Commit**

```bash
git add ui/src/LandingPage.system.test.ts
git commit -m "Add landing page live browser tests"
```

---

## Task 3: Catalog Version Detail Tests

**Files:**
- Create: `ui/src/CatalogVersionDetail.system.test.ts`

Tests: inline edit (label, description), BOM tab (pins visible, version dropdown, Add Pin modal), stage guards (testing CV hides controls for Admin, shows for SuperAdmin), transitions tab, diagram tab.

- [ ] **Step 1: Write CV detail tests**

This test file creates a CV via API, navigates to its detail page, and tests all tabs and interactions. It should cover:

1. Overview tab: version label, description, lifecycle stage badge
2. Inline edit: click Edit → TextInput appears → Cancel restores, Save calls API
3. BOM tab: pinned entity types listed, version dropdown (development CV), Add Pin button, Remove button
4. BOM tab with different roles: RO hides controls, Admin on testing hides controls, SuperAdmin on testing shows controls
5. Transitions tab: shows lifecycle history
6. Diagram tab: renders entity type diagram

Create test data via API in `beforeAll`:
- Create entity type `E2E_CVDetail_ET` with an attribute
- Create enum `E2E_CVDetail_Enum`
- Create CV `E2E_cvdetail-v1` with pin to the entity type

Test the BOM inline edit, Add Pin modal, stage guards by promoting/demoting the CV.

- [ ] **Step 2: Run tests**

Run: `cd ui && npx vitest run --config vitest.system.config.ts src/CatalogVersionDetail.system.test.ts`

- [ ] **Step 3: Commit**

```bash
git add ui/src/CatalogVersionDetail.system.test.ts
git commit -m "Add catalog version detail live browser tests"
```

---

## Task 4: Catalog List and Detail Tests

**Files:**
- Create: `ui/src/CatalogDetail.system.test.ts`

Tests: catalog creation, catalog detail page (instance CRUD, containment, references, validation, publish/unpublish, copy), inline edit description, CV selector.

- [ ] **Step 1: Write catalog tests**

Create test data via API in `beforeAll`:
- Entity types with containment association
- CV with pins
- Catalog pinned to CV with instances

Test flows:
1. Catalog list: create catalog, see it in list, delete it
2. Catalog detail: navigate, see entity type tabs, instance table
3. Instance CRUD: create instance, edit, delete
4. Containment: create contained instance, verify parent chain
5. Validation: click Validate, see results
6. Publish/unpublish: publish catalog, verify write protection, unpublish
7. Inline edit description: click Edit → change → Save
8. CV selector: change CV (re-pin), verify validation resets to draft

- [ ] **Step 2: Run tests**

Run: `cd ui && npx vitest run --config vitest.system.config.ts src/CatalogDetail.system.test.ts`

- [ ] **Step 3: Commit**

```bash
git add ui/src/CatalogDetail.system.test.ts
git commit -m "Add catalog detail live browser tests"
```

---

## Task 5: Data Viewer Tests

**Files:**
- Create: `ui/src/DataViewer.system.test.ts`

Tests: operational data viewer — tree browser, instance detail panel, containment navigation, reference navigation, model diagram tab.

- [ ] **Step 1: Write data viewer tests**

Uses existing catalog data (or creates test data). Tests:
1. Navigate to data viewer from catalog detail ("Open in Data Viewer →")
2. Containment tree loads with root entity types
3. Expand tree node → children appear
4. Click instance → detail panel shows attributes
5. Breadcrumb navigation (parent chain)
6. Reference links navigate to target instance
7. Model Diagram tab renders
8. Validate button triggers validation

- [ ] **Step 2: Run tests**

Run: `cd ui && npx vitest run --config vitest.system.config.ts src/DataViewer.system.test.ts`

- [ ] **Step 3: Commit**

```bash
git add ui/src/DataViewer.system.test.ts
git commit -m "Add data viewer live browser tests"
```

---

## Task 6: Security Flow Tests

**Files:**
- Create: `ui/src/SecurityFlows.system.test.ts`

Cross-cutting security tests that verify role enforcement and stage guards across pages.

- [ ] **Step 1: Write security tests**

Tests:
1. RO user: no Create/Edit/Delete buttons on any page
2. Published catalog: Admin sees no edit controls, no Validate; SuperAdmin sees all
3. Testing CV: Admin sees no edit/pin controls; SuperAdmin sees all
4. Production CV: nobody sees edit/pin controls (not even SuperAdmin)
5. Write-protected routes via API: PUT/DELETE on published catalog as RW returns 403
6. Stage guard via API: AddPin on production CV returns 400

- [ ] **Step 2: Run tests**

Run: `cd ui && npx vitest run --config vitest.system.config.ts src/SecurityFlows.system.test.ts`

- [ ] **Step 3: Commit**

```bash
git add ui/src/SecurityFlows.system.test.ts
git commit -m "Add security flow live browser tests"
```

---

## Task 7: Makefile Target and Documentation

**Files:**
- Modify: `Makefile` — add `test-e2e` target
- Modify: `docs/test-plan.md` — document live browser tests

- [ ] **Step 1: Add Makefile target**

Add a `test-e2e` target that runs all system tests (not included in `test-all`):

```makefile
test-e2e:
	cd "$(PROJECT_ROOT)ui" && npx vitest run --config vitest.system.config.ts
```

Note: `test-system` already exists and does the same thing. `test-e2e` is an alias for clarity. Consider renaming `test-system` to `test-e2e` or keeping both.

- [ ] **Step 2: Update test plan docs**

Add a section to `docs/test-plan.md` documenting the live browser test structure, how to run them, and what they cover.

- [ ] **Step 3: Run all system tests to verify nothing broke**

Run: `make -f /home/jsalomon/src/pc-asset-hub/Makefile test-system`
Expected: all system tests pass (existing 30 + new tests)

- [ ] **Step 4: Commit**

```bash
git add Makefile docs/test-plan.md
git commit -m "Add test-e2e Makefile target, update test plan docs"
```

---

## Task 8: Install Playwright for CI/Host

**Files:**
- Create: `scripts/install-playwright.sh` — one-liner to install browser deps

- [ ] **Step 1: Create install script**

```bash
#!/usr/bin/env bash
# Install Playwright browsers for system tests.
# Only needed on the host machine (not in containers).
set -euo pipefail
cd "$(dirname "$0")/../ui"
npx playwright install chromium
echo "Playwright chromium installed. Run: make test-e2e"
```

- [ ] **Step 2: Commit**

```bash
chmod +x scripts/install-playwright.sh
git add scripts/install-playwright.sh
git commit -m "Add Playwright install script for live browser tests"
```

---

## Test Count Estimate

| File | Tests | Coverage |
|------|-------|----------|
| `App.system.test.ts` (existing) | 30 | Entity types, enums, CVs, attributes, associations |
| `LandingPage.system.test.ts` | ~4 | Landing page navigation, role selector |
| `CatalogVersionDetail.system.test.ts` | ~12 | CV detail: overview, inline edit, BOM, stage guards, diagram |
| `CatalogDetail.system.test.ts` | ~15 | Catalog: list, detail, instance CRUD, containment, validation, publish |
| `DataViewer.system.test.ts` | ~8 | Operational viewer: tree, detail panel, refs, breadcrumbs |
| `SecurityFlows.system.test.ts` | ~10 | Role enforcement, stage guards, published protection |
| **Total** | **~79** | All pages and features |

## Key Patterns

- **Test data prefix:** All test data uses `E2E_` prefix for reliable cleanup
- **API setup, UI verification:** Create data via API in `beforeAll`, verify via browser, clean up in `afterAll`
- **Role switching:** Use `setRole(pg, 'SuperAdmin')` helper — note role resets on page navigation
- **No cross-test dependencies:** Each test file is self-contained with its own setup/teardown
- **Environment variables:** `UI_URL` and `API_URL` can be overridden for non-default deployments
