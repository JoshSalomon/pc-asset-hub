---
id: "mem_89515eca"
topic: "PC Asset Hub - complete testing strategy across all tiers"
tags:
  - testing
  - vitest
  - playwright
  - patternfly
  - coverage
  - jsdom
  - browser-mode
  - go-testing
  - system-tests
phase: 0
difficulty: 0.8
created_at: "2026-02-16T15:40:34.480514+00:00"
created_session: 3
---
## Project Testing Architecture

4-tier testing: Go backend, UI unit (jsdom), UI browser (Vitest Browser Mode), UI system (Playwright direct). Total: 113 UI tests + ~250 Go tests.

### Tier 1: Go Backend (`make test`)

- `go test ./...` with ~80.7% coverage
- Service tests: mock repos from `internal/domain/repository/mocks/`
- API handler tests: mock services + httptest + Echo
- Repo tests: in-memory SQLite via `testutil.NewTestDB(t)`
- Error branches: `closedDB(t)` helper closes DB to trigger `result.Error` paths
- Operator controller: `fake.NewClientBuilder()` from controller-runtime
- Build tags: `//go:build postgres_integration`, `//go:build e2e`

### Tier 2: UI Unit — jsdom (`npm test`, 55 tests)

- Config: `vite.config.ts`, environment jsdom
- PatternFly 6 interactive tests TIMEOUT in jsdom — only rendering/state tests work
- Required mocks in `src/test/setup.ts`: ResizeObserver, IntersectionObserver, matchMedia, scrollIntoView
- Must exclude browser files: `exclude: ['src/**/*.browser.test.{ts,tsx}']`

### Tier 3: UI Browser — Vitest Browser Mode (`npm run test:browser`, 45 tests, 97.95% coverage)

- Config: `vitest.browser.config.ts`, real Chromium via Playwright
- All PatternFly interactive behavior works (click, type, modal, tabs, dropdown)
- Uses `vitest-browser-react` for render, `page` from `vitest/browser` for locators
- Vitest 4 API: `import { playwright } from '@vitest/browser-playwright'` then `provider: playwright()` (NOT string)
- Modal buttons: `page.getByRole('dialog').getByRole('button', { name: 'Create' })`
- api/client.ts tested without module mock via `vi.stubGlobal('fetch', mockFetch)`
- Include pattern needs `{ts,tsx}`: `include: ['src/**/*.browser.test.{ts,tsx}']`
- CANNOT navigate to external URLs (breaks WebSocket) — use Tier 4 for that

### Tier 4: UI System — Playwright Direct (`npm run test:system`, 13 tests)

- Config: `vitest.system.config.ts`, runs in Node
- Prerequisite: live kind cluster (`./scripts/kind-deploy.sh up`)
- Launches `chromium.launch({ headless: true })`, navigates to `localhost:30000`
- Assertions via `locator.waitFor({ state: 'visible' })` NOT `expect().toBeVisible()` (Vitest expect lacks Playwright matchers)
- API setup/teardown via Node `fetch()` directly
- Unique names with `Date.now()` for test isolation
- Strict mode: PatternFly Tabs renders hidden panels — use `.first()` for duplicates

### Commands

```bash
make test                              # Go backend
cd ui && npm test                      # jsdom unit (55 tests)
cd ui && npm run test:browser          # Chromium component (45 tests)
cd ui && npm run test:system           # System against live cluster (13 tests)
cd ui && npm run test:all              # jsdom + browser combined
cd ui && npm run test:browser -- --coverage  # 97.95% statements
```

### Key Pitfalls

- PatternFly 6 + jsdom: interactive tests timeout. Use Browser Mode.
- Vitest Browser Mode cannot `page.goto()` external URLs. Use Playwright direct.
- `vi.stubGlobal('fetch', mockFetch)` tests real client code without module mock.
- Modal buttons: target via `page.getByRole('dialog').getByRole('button', ...)`.
- Duplicate DOM text from PatternFly Tabs hidden panels: use `.first()`.
- Go cross-package coverage: use `-coverpkg=./internal/...` for aggregate numbers.
- Go error branches in GORM repos: use `closedDB(t)` to trigger non-ErrRecordNotFound errors.

