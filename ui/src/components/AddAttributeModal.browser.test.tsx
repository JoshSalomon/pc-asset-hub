import { useState } from 'react'
import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach } from 'vitest'
import { page } from 'vitest/browser'
import AddAttributeModal from './AddAttributeModal'
import type { TypeDefinition } from '../types'

const mockTypeDefinitions: TypeDefinition[] = [
  { id: 'td-string', name: 'string', base_type: 'string', system: true, latest_version: 1, latest_version_id: 'tdv-string', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td-number', name: 'number', base_type: 'number', system: true, latest_version: 1, latest_version_id: 'tdv-number', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td1', name: 'Colors', base_type: 'enum', system: false, latest_version: 1, latest_version_id: 'tdv-colors', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  { id: 'td2', name: 'Sizes', base_type: 'enum', system: false, latest_version: 1, latest_version_id: 'tdv-sizes', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
]

function renderModal(overrides: Partial<React.ComponentProps<typeof AddAttributeModal>> = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    onSubmit: vi.fn().mockResolvedValue(undefined),
    typeDefinitions: mockTypeDefinitions,
    error: null,
    ...overrides,
  }
  return { ...render(<AddAttributeModal {...props} />), props }
}

beforeEach(() => {
  vi.clearAllMocks()
})

// T-20.30: Renders modal with title
test('T-20.30: AddAttributeModal renders title', async () => {
  renderModal()
  await expect.element(page.getByText('Add Attribute')).toBeVisible()
})

// T-20.31: Name field is required
test('T-20.31: AddAttributeModal has name field', async () => {
  renderModal()
  const nameInput = page.getByRole('textbox', { name: 'Name' })
  await expect.element(nameInput).toBeVisible()
})

// T-20.32: Add button disabled when name empty
test('T-20.32: AddAttributeModal add disabled when name empty', async () => {
  renderModal()
  const addBtn = page.getByRole('button', { name: 'Add' })
  await expect.element(addBtn).toHaveAttribute('disabled')
})

// T-20.33: Type shows placeholder when no type selected
test('T-20.33: AddAttributeModal type defaults to select placeholder', async () => {
  renderModal()
  await expect.element(page.getByText('Select type...')).toBeVisible()
})

// T-20.34: Shows error
test('T-20.34: AddAttributeModal shows error', async () => {
  renderModal({ error: 'Duplicate name' })
  await expect.element(page.getByText('Duplicate name')).toBeVisible()
})

// T-20.35: Calls onSubmit with form values
test('T-20.35: AddAttributeModal calls onSubmit', async () => {
  const { props } = renderModal()
  // Fill name
  await page.getByRole('textbox', { name: 'Name' }).fill('hostname')
  // Select a type — open the type selector and pick string
  await page.getByText('Select type...').click()
  await page.getByText('string').click()
  // Click Add
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    name: 'hostname',
    typeDefinitionVersionId: 'tdv-string',
    required: false,
  }))
})

test('AddAttributeModal does not submit when type has no latest_version_id', async () => {
  // Render with a type definition that has empty latest_version_id
  const badTypes: TypeDefinition[] = [
    { id: 'td-broken', name: 'broken', base_type: 'string', system: false, latest_version: 1, latest_version_id: '', created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' },
  ]
  const onSubmit = vi.fn()
  render(
    <AddAttributeModal isOpen onClose={vi.fn()} onSubmit={onSubmit} error={null} typeDefinitions={badTypes} />
  )
  await page.getByRole('textbox', { name: 'Name' }).fill('test')
  await page.getByText('Select type...').click()
  await page.getByText('broken (string)').click()
  await page.getByRole('button', { name: 'Add' }).click()
  expect(onSubmit).not.toHaveBeenCalled()
})

// T-20.36: Cancel calls onClose
test('T-20.36: AddAttributeModal cancel calls onClose', async () => {
  const { props } = renderModal()
  await page.getByRole('button', { name: 'Cancel' }).click()
  expect(props.onClose).toHaveBeenCalled()
})

// T-20.37: Required checkbox toggles
test('T-20.37: AddAttributeModal required checkbox', async () => {
  const { props } = renderModal()
  await page.getByRole('textbox', { name: 'Name' }).fill('test')
  // Select a type
  await page.getByText('Select type...').click()
  await page.getByText('string').click()
  // Click the Required checkbox
  await page.getByRole('checkbox').click()
  await page.getByRole('button', { name: 'Add' }).click()
  expect(props.onSubmit).toHaveBeenCalledWith(expect.objectContaining({
    required: true,
  }))
})

// Line 63: if (!td) return — guard in handleSubmit after find()
// UNREACHABLE: The Add button is disabled when !selectedTdId (line 130),
// so handleSubmit can only run when selectedTdId is set. Since selectedTdId
// was set via the Select dropdown which only shows typeDefinitions items,
// find() always succeeds. This is a defensive guard that cannot be triggered
// through the component's UI.

// T-20.38: Reopen resets form state (I5 — useEffect reset on isOpen)
test('T-20.38: AddAttributeModal reopen resets name field', async () => {
  // Use a wrapper that exposes force close/open functions to simulate the
  // hook's setAddAttrOpen(false) path (isOpen toggled without handleClose)
  let forceClose: () => void = () => {}
  let forceOpen: () => void = () => {}
  function Wrapper() {
    const [open, setOpen] = useState(true)
    forceClose = () => setOpen(false)
    forceOpen = () => setOpen(true)
    return (
      <AddAttributeModal
        isOpen={open}
        onClose={() => setOpen(false)}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        typeDefinitions={mockTypeDefinitions}
        error={null}
      />
    )
  }
  render(<Wrapper />)
  await expect.element(page.getByText('Add Attribute')).toBeVisible()

  // Fill in the name
  await page.getByRole('textbox', { name: 'Name' }).fill('stale-attr')
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('stale-attr')

  // Close by directly setting isOpen=false (simulates hook path)
  forceClose()
  // Wait for the modal to close
  await expect.element(page.getByText('Add Attribute')).not.toBeInTheDocument()

  // Reopen
  forceOpen()
  await expect.element(page.getByText('Add Attribute')).toBeVisible()

  // Name should be reset
  await expect.element(page.getByRole('textbox', { name: 'Name' })).toHaveValue('')
})

// TD-109 / T-28.07: System types show just name, not redundant "name (name)"
test('T-28.07: system types show just name, not redundant base_type', async () => {
  renderModal()
  await page.getByText('Select type...').click()
  // Inspect the rendered menu item text
  const menuItems = document.querySelectorAll('[class*="menu__item"] [class*="menu__item-text"]')
  const texts = Array.from(menuItems).map(el => el.textContent)
  // System types should show just name
  expect(texts).toContain('string')
  expect(texts).toContain('number')
  // Custom types should show "name (base_type)"
  expect(texts).toContain('Colors (enum)')
  expect(texts).toContain('Sizes (enum)')
})

// TD-95 / T-28.17: Helper text shown when string type selected
test('T-28.17: helper text shown for string-based type', async () => {
  renderModal()
  await page.getByText('Select type...').click()
  await page.getByText('string', { exact: true }).click()
  await expect.element(page.getByText(/multiline/i)).toBeVisible()
})

// TD-97 / T-28.05: Type selector has maxHeight for scrollability
test('T-28.05: type selector dropdown has max height', async () => {
  renderModal()
  await page.getByText('Select type...').click()
  const scrollContainer = document.querySelector('[data-testid="type-select-scroll"]')
  expect(scrollContainer).not.toBeNull()
  const style = window.getComputedStyle(scrollContainer!)
  expect(style.maxHeight).toBe('200px')
  expect(style.overflow).toBe('auto')
})

// TD-97 / T-28.06: Type selector supports typeahead filtering
test('T-28.06: type selector has filter input', async () => {
  renderModal()
  await page.getByText('Select type...').click()
  await expect.element(page.getByPlaceholder('Filter types...')).toBeVisible()
})

// Coverage: TypeDefinitionSelector onChange — typing in filter input filters options
test('type selector filter input filters displayed types', async () => {
  renderModal()
  await page.getByText('Select type...').click()
  const filterInput = page.getByPlaceholder('Filter types...')
  await filterInput.fill('Col')
  // "Colors" should be visible, "Sizes" should be hidden
  await expect.element(page.getByText('Colors (enum)')).toBeVisible()
  // "Sizes" should not appear since "Col" doesn't match "Sizes"
  const menuItems = document.querySelectorAll('[class*="menu__item"] [class*="menu__item-text"]')
  const texts = Array.from(menuItems).map(el => el.textContent)
  expect(texts).not.toContain('Sizes (enum)')
})

// Coverage: TypeDefinitionSelector onOpenChange — clicking outside closes dropdown
test('type selector clicking outside closes dropdown via onOpenChange', async () => {
  renderModal()
  // Open the type selector
  await page.getByText('Select type...').click()
  await expect.element(page.getByPlaceholder('Filter types...')).toBeVisible()
  // Type a filter value
  const filterInput = page.getByPlaceholder('Filter types...')
  await filterInput.fill('Col')
  // Click on the modal title to trigger PF6's outside-click handler → onOpenChange(false)
  await page.getByText('Add Attribute').click()
  // The filter input should disappear (dropdown closed)
  await expect.element(page.getByPlaceholder('Filter types...')).not.toBeInTheDocument()
})
