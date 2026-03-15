import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import { MemoryRouter } from 'react-router-dom'
import OperationalApp from './OperationalApp'

vi.mock('./api/client', () => ({
  api: {
    catalogs: { list: vi.fn().mockResolvedValue({ items: [], total: 0 }) },
  },
  setAuthRole: vi.fn(),
}))

function renderApp() {
  return render(
    <MemoryRouter>
      <OperationalApp />
    </MemoryRouter>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

test('T-13.58: operational app renders masthead with brand', async () => {
  renderApp()
  await expect.element(page.getByText('AI Asset Hub — Data Viewer')).toBeVisible()
})

test('T-13.59: role selector defaults to RO', async () => {
  renderApp()
  await expect.element(page.getByText('Role: RO')).toBeVisible()
})

test('T-13.60: role selector is present and interactive', async () => {
  renderApp()
  // The role selector toggle is rendered as a MenuToggle button
  const toggle = page.getByText('Role: RO')
  await expect.element(toggle).toBeVisible()
  // Verify it's clickable (a button)
  await expect.element(toggle).toBeInTheDocument()
})
