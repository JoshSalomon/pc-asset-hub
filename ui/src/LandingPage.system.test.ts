// System tests for LandingPage — run against a live deployment (kind cluster).
// Uses Playwright directly to launch a real browser, navigate to the
// deployed UI, and interact with it. No mocks.
//
// Prerequisites:
//   - kind cluster running: ./scripts/kind-deploy.sh up
//   - UI at http://localhost:30000, API at http://localhost:30080
//
// Run:
//   npm run test:system -- src/LandingPage.system.test.ts

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

afterAll(async () => { await teardownBrowser(browser) })

test('landing page loads with title and sections', async () => {
  await pg.goto(UI_URL)
  await visible(pg.getByRole('heading', { name: 'AI Asset Hub', exact: true }))
  await visible(pg.getByRole('heading', { name: 'Schema Management' }))
  await visible(pg.getByRole('heading', { name: 'Catalogs' }))
})

test('schema management card navigates to /schema', async () => {
  await pg.goto(UI_URL)
  await visible(pg.getByText('Entity Types & Model'))
  await pg.getByText('Entity Types & Model').click()

  // Use pg.url() instead of toHaveURL since we're not using @playwright/test
  await pg.waitForTimeout(500) // Give navigation time to complete
  expect(pg.url()).toMatch(/\/schema/)
  await visible(pg.getByRole('tab', { name: 'Entity Types' }))
})

test('catalog section shows on landing page', async () => {
  await pg.goto(UI_URL)
  // Wait for catalogs section to load (shows catalogs or empty state)
  await pg.waitForTimeout(1000)
  const bodyText = await pg.textContent('body')
  // Either catalogs are displayed or the empty state shows
  const hasCatalogs = bodyText?.includes('Version:')
  const hasEmpty = bodyText?.includes('No catalogs available')
  expect(hasCatalogs || hasEmpty).toBe(true)
})

test('role selector shows all 4 roles', async () => {
  await pg.goto(UI_URL)
  const roleBtn = pg.locator('button').filter({ hasText: /Role:/ }).first()
  await roleBtn.click()
  await visible(pg.getByRole('option', { name: 'RO', exact: true }))
  await visible(pg.getByRole('option', { name: 'RW', exact: true }))
  await visible(pg.getByRole('option', { name: 'Admin', exact: true }))
  await visible(pg.getByRole('option', { name: 'SuperAdmin', exact: true }))
  // Close dropdown
  await pg.keyboard.press('Escape')
})
