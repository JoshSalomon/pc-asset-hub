---
id: "mem_3c8faa5c"
topic: "Browser test coverage strategy — how to maximize coverage on every feature"
tags:
  - testing
  - coverage
  - browser-tests
  - best-practices
phase: 0
difficulty: 0.7
created_at: "2026-03-16T17:25:32.656628+00:00"
created_session: 16
---
## Coverage Gap Root Causes

The biggest browser coverage gap is almost always `api/client.ts`. Page-level browser tests mock the API client (`vi.mock`), so the real `client.ts` code NEVER executes. The mock replaces the module entirely. This means every new API method added to `client.ts` drops the overall coverage percentage.

## The Fix: Direct Client Tests

`client.browser.test.ts` uses `vi.stubGlobal('fetch', mockFetch)` — it stubs the global `fetch` function, so the REAL client code executes (URL construction, method selection, body serialization, response parsing). Every new API method must have a corresponding test in this file.

Pattern:
```typescript
test('catalogs.publish sends POST to /publish', async () => {
  mockFetch.mockReturnValue(jsonResponse({ status: 'published' }))
  await api.catalogs.publish('my-cat')
  const [url, opts] = mockFetch.mock.calls[0]
  expect(url).toContain('/catalogs/my-cat/publish')
  expect(opts.method).toBe('POST')
})
```

This single test covers 2-3 statements in client.ts (the function body + fetchJSON call).

## Coverage Checklist for Every Feature

1. **New API methods in `client.ts`** → add direct tests in `client.browser.test.ts` using `mockFetch`
2. **New UI components** → test rendering with various props (including error/empty states)
3. **New hooks** → test in a dedicated `*.browser.test.tsx` with a TestComponent wrapper, cover:
   - Happy path
   - Error path (reject with `new Error`)
   - Non-Error rejection (reject with string) for ternary branches
   - Missing/undefined parameters (early returns)
   - Optional callbacks (with and without)
4. **New conditional rendering** (badges, buttons, banners) → test with mock data that triggers each branch:
   - `published: true` vs `published: false`
   - Each role (RO, RW, Admin, SuperAdmin)
   - Each validation status (draft, valid, invalid)
5. **Error handlers in onClick** → test with mockRejectedValue to cover catch blocks
6. **Mock data in existing tests** → update to include new fields (e.g., add `published: true` to one mock catalog in list tests)

## Tools

- `scripts/analyze-coverage.sh` — shows all files below 100% with statement counts
- `scripts/analyze-coverage.sh <pattern>` — filter by filename pattern
- Coverage summary JSON: `ui/coverage/coverage-summary.json`

## Common Pitfalls

- Page tests mock the API → client.ts stays uncovered. Always add client.browser.test.ts tests.
- `getByText('valid')` matches 'Validate' button substring → use `{ exact: true }`
- PatternFly Tabs renders ALL tab content on mount → call `setAuthRole(role)` in load functions
- `vi.mock` must list ALL methods used by the component, including new ones (e.g., add `publish: vi.fn()` when adding publish feature)
