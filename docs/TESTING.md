# PC Asset Hub — Complete Testing Strategy

## Overview

4-tier testing strategy: Go backend, UI unit (jsdom), UI browser (Vitest Browser Mode), and UI system tests (Playwright direct). Total: 113 UI tests + ~250 Go tests.

## Tier 1: Go Backend Tests

```bash
make test          # All Go unit/integration tests
make coverage      # With coverage report (~80.7%)
make test-postgres # PostgreSQL integration (needs running PG)
make e2e-test      # E2E against live kind cluster
```

**Patterns:**
- Service tests use mock repos from `internal/domain/repository/mocks/`
- API handler tests use mock services + `httptest` + Echo context
- Repository tests use `testutil.NewTestDB(t)` (in-memory SQLite with auto-migrate)
- Error branch tests use `closedDB(t)` — closes DB connection to trigger `result.Error` paths
- Operator controller tests use `fake.NewClientBuilder()` from controller-runtime
- Build-tag guarded: `//go:build postgres_integration`, `//go:build e2e`

## Tier 2: UI Unit Tests — jsdom

```bash
cd ui && npm test   # 55 tests, ~50% coverage
```

**Config:** `vite.config.ts`

**Key limitation:** PatternFly 6 components (Tabs, Select, Modal, Button) use browser APIs that jsdom doesn't implement. Interactive tests (click, type, tab switch) **will timeout** in jsdom.

**What works:** Rendering, data display, empty states, error alerts, component props, table content.

**Required mocks in `src/test/setup.ts`:**
```typescript
window.ResizeObserver = MockResizeObserver     // PatternFly layout
window.IntersectionObserver = MockIntersectionObserver  // PatternFly visibility
window.matchMedia = vi.fn().mockImplementation(...)     // Responsive components
window.HTMLElement.prototype.scrollIntoView = vi.fn()   // Tabs overflow
```

**Must exclude browser test files** in `vite.config.ts`:
```typescript
exclude: ['src/**/*.browser.test.{ts,tsx}', 'node_modules/**']
```

## Tier 3: UI Browser Tests — Vitest Browser Mode

```bash
cd ui && npm run test:browser   # 45 tests, 97.95% coverage
```

**Config:** `vitest.browser.config.ts`

Tests run in real Chromium via Vitest Browser Mode. All PatternFly interactive behavior works because it's a real browser.

**Key patterns:**
```typescript
import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'

test('example', async () => {
  render(<App />)
  await expect.element(page.getByText('AI Asset Hub')).toBeVisible()
  await page.getByRole('button', { name: 'Create' }).click()
  // Modal buttons via dialog role:
  await page.getByRole('dialog').getByRole('button', { name: 'Create' }).click()
})
```

**Vitest 4 API (important — changed from v3):**
```typescript
// Provider uses factory import, NOT a string
import { playwright } from '@vitest/browser-playwright'
// In config:
provider: playwright()   // NOT provider: 'playwright'
```

**Testing api/client.ts without mocking the module:**
```typescript
vi.stubGlobal('fetch', mockFetch)  // Intercept fetch at global level
```

**Cannot navigate to external URLs** — `page.goto()` doesn't exist on Vitest's page locator, and navigating the underlying browser away breaks the WebSocket connection to the test runner.

## Tier 4: UI System Tests — Playwright Direct

```bash
cd ui && npm run test:system   # 13 tests against live cluster
```

**Config:** `vitest.system.config.ts`

**Prerequisites:** Kind cluster running (`./scripts/kind-deploy.sh up`)

Tests launch Playwright's Chromium directly from Node and navigate to the deployed UI at `http://localhost:30000`.

**Key patterns:**
```typescript
import { chromium, type Browser, type Page, type Locator } from 'playwright'

// Vitest's expect() does NOT have Playwright matchers.
// Use waitFor instead:
async function visible(locator: Locator) {
  await locator.waitFor({ state: 'visible', timeout: 10000 })
}

// API calls for setup/teardown run server-side (Node fetch):
const res = await fetch(`${API_URL}/api/meta/v1/entity-types`, { ... })

// Each test creates unique names for isolation:
const name = `SysTest_${Date.now()}`
```

**Strict mode violations:** PatternFly Tabs renders all tab panels even when hidden. Locators like `getByText('Total: 0')` may match multiple elements. Use `.first()` or more specific selectors.

---

## Phase C: Entity Type Management Tests

### High-Level Strategy

#### Backend (Go) — Handler Tests

For each new handler (attribute, association, enum, version history), test:
- **Happy path**: valid request → correct response + status code
- **RBAC**: RO user → 403 on write endpoints, 200 on read endpoints
- **Validation**: missing required fields → 400
- **Not found**: nonexistent entity type / enum → 404
- **Conflict**: duplicate attribute name → 409
- **Domain errors**: cycle detection (containment association) → 422, enum referenced by attribute → 422

#### Frontend (UI) — Browser Tests (Vitest Browser Mode + Chromium)

- **Delete confirmation**: click Delete → modal appears → Cancel does nothing → Confirm deletes
- **Entity type detail page**: navigate to detail → see attributes, associations, versions
- **Attribute management**: add attribute → appears in table → remove → gone → reorder
- **Association management**: add association → appears → remove → cycle detection error shown
- **Enum management**: create enum → add values → use in attribute → delete referenced enum shows error
- **Version history**: view versions → compare two → see diff
- **RBAC**: RO user sees data but no add/remove/reorder controls

#### System Tests (Playwright against live cluster)

- End-to-end workflow: create entity type → add attributes → add association → create catalog version → verify via API

---

### Detailed Test Plan

#### Attribute Handler Tests (T-C.01 through T-C.08)

| ID | Test | Method | Expected |
|----|------|--------|----------|
| T-C.01 | List attributes for entity type | GET /entity-types/:id/attributes | 200 + attribute array |
| T-C.02 | Add string attribute | POST /entity-types/:id/attributes | 201 + new version |
| T-C.03 | Add enum attribute with valid enum_id | POST | 201 |
| T-C.04 | Add attribute missing name → 400 | POST | 400 |
| T-C.05 | Add duplicate attribute name → 409 | POST | 409 |
| T-C.06 | Remove attribute by name | DELETE /entity-types/:id/attributes/:name | 204 |
| T-C.07 | Reorder attributes | PUT /entity-types/:id/attributes/reorder | 200 |
| T-C.08 | Add attribute as RO → 403 | POST | 403 |

#### Association Handler Tests (T-C.09 through T-C.14)

| ID | Test | Expected |
|----|------|----------|
| T-C.09 | List associations | 200 + association array |
| T-C.10 | Create containment association | 201 + new version |
| T-C.11 | Create directional association | 201 |
| T-C.12 | Create containment cycle → 422 | 422 cycle detected |
| T-C.13 | Delete association | 204 |
| T-C.14 | Create association as RO → 403 | 403 |

#### Enum Handler Tests (T-C.15 through T-C.25)

| ID | Test | Expected |
|----|------|----------|
| T-C.15 | List enums | 200 + enum array |
| T-C.16 | Create enum with values | 201 |
| T-C.17 | Create enum missing name → 400 | 400 |
| T-C.18 | Create duplicate enum name → 409 | 409 |
| T-C.19 | Get enum by ID | 200 |
| T-C.20 | Update enum name | 200 |
| T-C.21 | Delete enum | 204 |
| T-C.22 | Delete referenced enum → 422 | 422 |
| T-C.23 | List enum values | 200 + value array |
| T-C.24 | Add enum value | 201 |
| T-C.25 | Reorder enum values | 200 |

#### Version History Handler Tests (T-C.26 through T-C.28)

| ID | Test | Expected |
|----|------|----------|
| T-C.26 | Get version history | 200 + version array |
| T-C.27 | Compare two versions | 200 + diff |
| T-C.28 | Compare nonexistent version → 404 | 404 |

#### RBAC Tests (T-C.53 through T-C.74)

Role hierarchy: RO (0) < RW (1) < Admin (2) < SuperAdmin (3). Write endpoints require Admin+. Read endpoints require any authenticated role.

**Attribute RBAC:**

| ID | Test | Role | Method | Expected |
|----|------|------|--------|----------|
| T-C.53 | RO can list attributes | RO | GET | 200 |
| T-C.54 | RW cannot add attribute | RW | POST | 403 |
| T-C.55 | Admin can add attribute | Admin | POST | 201 |
| T-C.56 | SuperAdmin can add attribute | SuperAdmin | POST | 201 |
| T-C.57 | RO cannot remove attribute | RO | DELETE | 403 |
| T-C.58 | RO cannot reorder attributes | RO | PUT reorder | 403 |

**Association RBAC:**

| ID | Test | Role | Method | Expected |
|----|------|------|--------|----------|
| T-C.59 | RO can list associations | RO | GET | 200 |
| T-C.60 | RW cannot create association | RW | POST | 403 |
| T-C.61 | Admin can create association | Admin | POST | 201 |
| T-C.62 | SuperAdmin can create association | SuperAdmin | POST | 201 |
| T-C.63 | RO cannot delete association | RO | DELETE | 403 |

**Enum RBAC:**

| ID | Test | Role | Method | Expected |
|----|------|------|--------|----------|
| T-C.64 | RO can list enums | RO | GET | 200 |
| T-C.65 | RO can get enum by ID | RO | GET :id | 200 |
| T-C.66 | RO can list enum values | RO | GET :id/values | 200 |
| T-C.67 | RW cannot create enum | RW | POST | 403 |
| T-C.68 | Admin can create enum | Admin | POST | 201 |
| T-C.69 | SuperAdmin can create enum | SuperAdmin | POST | 201 |
| T-C.70 | RO cannot update enum | RO | PUT | 403 |
| T-C.71 | RO cannot delete enum | RO | DELETE | 403 |
| T-C.72 | RO cannot add enum value | RO | POST values | 403 |

**Version History RBAC** — read-only endpoints, all roles can access:

| ID | Test | Role | Method | Expected |
|----|------|------|--------|----------|
| T-C.73 | RO can list versions | RO | GET | 200 |
| T-C.74 | RO can compare versions | RO | GET diff | 200 |

#### UI Browser Tests — Delete Confirmation (T-C.29 through T-C.31)

| ID | Test |
|----|------|
| T-C.29 | Click Delete → confirmation modal shows entity type name |
| T-C.30 | Cancel confirmation → no deletion |
| T-C.31 | Confirm deletion → API called, entity removed from list |

#### UI Browser Tests — Entity Type Detail Page (T-C.32 through T-C.45)

| ID | Test |
|----|------|
| T-C.32 | Navigate to detail page → shows name, description, version |
| T-C.33 | Attributes tab lists attributes |
| T-C.34 | Add string attribute via modal |
| T-C.35 | Add enum attribute — enum selector shown when type=enum |
| T-C.36 | Remove attribute |
| T-C.37 | Reorder attributes with up/down buttons |
| T-C.38 | Associations tab lists associations |
| T-C.39 | Add containment association via modal |
| T-C.40 | Cycle detection error displayed (422) |
| T-C.41 | Remove association |
| T-C.42 | Version history tab shows versions |
| T-C.43 | Compare two versions shows diff |
| T-C.44 | Copy entity type via modal |
| T-C.45 | RO role hides add/remove controls |

#### UI Browser Tests — Enum Management (T-C.46 through T-C.52)

| ID | Test |
|----|------|
| T-C.46 | Enum list page shows enums |
| T-C.47 | Create enum with initial values |
| T-C.48 | Navigate to enum detail |
| T-C.49 | Add value to enum |
| T-C.50 | Remove value from enum |
| T-C.51 | Reorder enum values |
| T-C.52 | Delete referenced enum shows error |

---

## Common Pitfalls

| Problem | Solution |
|---------|----------|
| PatternFly interactive tests timeout in jsdom | Use Vitest Browser Mode (Tier 3) |
| `page.goto()` in Vitest Browser Mode | Can't — use Playwright direct (Tier 4) for external URLs |
| Multiple "Total: N" matches | PatternFly Tabs renders hidden panels — use `.first()` |
| Modal Create button conflicts with toolbar Create button | Target via `page.getByRole('dialog').getByRole('button', ...)` |
| `vitest/browser` import fails outside browser | That module only works inside Vitest Browser Mode |
| `provider: 'playwright'` doesn't work | Vitest 4 changed API — use `playwright()` factory from `@vitest/browser-playwright` |
| Error branch coverage in Go repos | Use `closedDB(t)` helper to trigger non-ErrRecordNotFound errors |
| Cross-package coverage in Go | Per-package coverage misses cross-package calls. Use `-coverpkg=./internal/...` for aggregate |

## Run Everything

```bash
# Backend
make test && make lint

# UI — all tiers
cd ui
npm test              # Tier 2: jsdom (55 tests)
npm run test:browser  # Tier 3: Chromium component tests (45 tests)
npm run test:system   # Tier 4: system tests against live cluster (13 tests)

# Coverage
npm run test:browser -- --coverage   # 97.95% statements
```
