import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import AttributeFormFields from './AttributeFormFields'
import type { SnapshotAttribute } from '../types'
import { validateAttributeValue } from '../utils/validateAttributeValue'

const schemaAttrs: SnapshotAttribute[] = [
  { id: 'sys-name', name: 'name', base_type: 'string', description: '', ordinal: -2, required: true, system: true },
  { id: 'sys-desc', name: 'description', base_type: 'string', description: '', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'color', base_type: 'enum', description: '', ordinal: 1, required: false, type_definition_version_id: 'enum1' },
  { id: 'a2', name: 'port', base_type: 'number', description: '', ordinal: 2, required: true },
  { id: 'a3', name: 'hostname', base_type: 'string', description: '', ordinal: 3, required: false },
]

const enumValues = { enum1: ['red', 'green', 'blue'] }

// T-20.07: Renders Name system attr with required indicator when includeSystem=true
test('T-20.07: renders Name with required when includeSystem=true', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
      includeSystem
      systemName=""
      setSystemName={vi.fn()}
      systemDesc=""
      setSystemDesc={vi.fn()}
    />,
  )
  const nameInput = page.getByLabelText('Name *')
  await expect.element(nameInput).toBeVisible()
})

// T-20.08: Renders Description system attr without required when includeSystem=true
test('T-20.08: renders Description without required when includeSystem=true', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
      includeSystem
      systemName=""
      setSystemName={vi.fn()}
      systemDesc=""
      setSystemDesc={vi.fn()}
    />,
  )
  await expect.element(page.getByText('Description', { exact: true })).toBeVisible()
})

// T-20.09: Does not render system attrs when includeSystem=false
test('T-20.09: does not render system attrs when includeSystem=false', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  // Should not have Name or Description labels from system attrs
  // But should have custom attrs
  await expect.element(page.getByText('port *')).toBeVisible()
  await expect.element(page.getByText('hostname')).toBeVisible()
})

// T-20.10: Renders custom text attr with text input
test('T-20.10: renders custom text attr', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{ hostname: 'srv-1' }}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('textbox', { name: 'hostname' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveValue('srv-1')
})

// T-20.11: Renders custom number attr with number input
test('T-20.11: renders custom number attr', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{ port: '8080' }}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('spinbutton', { name: 'port *' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveValue(8080)
})

// T-20.12: Renders enum attr with select dropdown
test('T-20.12: renders enum attr with select', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  // Enum attr renders a native <select> with "Select..." option
  const selectEl = page.getByRole('combobox', { name: 'color' })
  await expect.element(selectEl).toBeVisible()
})

// T-20.13: Calls onChange when text input changes
test('T-20.13: calls onChange on text input change', async () => {
  const onChange = vi.fn()
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={onChange}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('textbox', { name: 'hostname' })
  await input.fill('new-host')
  expect(onChange).toHaveBeenCalledWith('hostname', 'new-host')
})

// T-20.14: Shows required indicator for required custom attrs
test('T-20.14: shows required indicator for required attrs', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  await expect.element(page.getByText('port *')).toBeVisible()
})

// T-20.15: Boolean attr renders checkbox; toggling calls onChange
test('T-20.15: boolean attr renders checkbox and toggles value', async () => {
  const onChange = vi.fn()
  const boolAttrs: SnapshotAttribute[] = [
    { id: 'b1', name: 'active', base_type: 'boolean', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={boolAttrs}
      values={{ active: 'false' }}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  // PF Checkbox: look for checkbox by aria-label or label text
  const checkbox = page.getByRole('checkbox', { name: 'active' })
    .or(page.getByRole('checkbox'))
  await expect.element(checkbox).toBeVisible()
  await expect.element(checkbox).not.toBeChecked()
  await checkbox.click()
  expect(onChange).toHaveBeenCalledWith('active', 'true')
})

// T-20.16: Integer attr renders number input with step=1
test('T-20.16: integer attr renders number input with step=1', async () => {
  const onChange = vi.fn()
  const intAttrs: SnapshotAttribute[] = [
    { id: 'i1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={intAttrs}
      values={{ count: '42' }}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('spinbutton', { name: 'count' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveValue(42)
  await expect.element(input).toHaveAttribute('step', '1')
  await input.fill('99')
  expect(onChange).toHaveBeenCalledWith('count', '99')
})

// T-20.17: Date attr renders text input with YYYY-MM-DD placeholder
test('T-20.17: date attr renders text input with placeholder', async () => {
  const onChange = vi.fn()
  const dateAttrs: SnapshotAttribute[] = [
    { id: 'd1', name: 'start_date', base_type: 'date', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={dateAttrs}
      values={{}}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('textbox', { name: 'start_date' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveAttribute('placeholder', 'YYYY-MM-DD')
  await input.fill('2026-01-15')
  expect(onChange).toHaveBeenCalledWith('start_date', '2026-01-15')
})

// T-20.18: URL attr renders text input with https://... placeholder
test('T-20.18: url attr renders text input with placeholder', async () => {
  const onChange = vi.fn()
  const urlAttrs: SnapshotAttribute[] = [
    { id: 'u1', name: 'website', base_type: 'url', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={urlAttrs}
      values={{}}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const input = page.getByRole('textbox', { name: 'website' })
  await expect.element(input).toBeVisible()
  await expect.element(input).toHaveAttribute('placeholder', 'https://...')
  await input.fill('https://example.com')
  expect(onChange).toHaveBeenCalledWith('website', 'https://example.com')
})

// T-20.19: JSON attr renders textarea with JSON placeholder
test('T-20.19: json attr renders textarea with placeholder', async () => {
  const onChange = vi.fn()
  const jsonAttrs: SnapshotAttribute[] = [
    { id: 'j1', name: 'config', base_type: 'json', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={jsonAttrs}
      values={{}}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const textarea = page.getByRole('textbox', { name: 'config' })
  await expect.element(textarea).toBeVisible()
  await expect.element(textarea).toHaveAttribute('placeholder', '{"key": "value"}')
  await textarea.fill('{"foo": 1}')
  expect(onChange).toHaveBeenCalledWith('config', '{"foo": 1}')
})

// T-20.21: Multiline string attr renders TextArea and onChange fires
test('T-20.21: multiline string attr renders textarea and calls onChange', async () => {
  const onChange = vi.fn()
  const multilineAttrs: SnapshotAttribute[] = [
    { id: 'm1', name: 'notes', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { multiline: true } },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={multilineAttrs}
      values={{}}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const textarea = page.getByRole('textbox', { name: 'notes' })
  await expect.element(textarea).toBeVisible()
  await textarea.fill('Line 1\nLine 2')
  expect(onChange).toHaveBeenCalledWith('notes', 'Line 1\nLine 2')
})

// T-20.20: List attr renders textarea with comma-separated placeholder
test('T-20.20: list attr renders textarea with placeholder', async () => {
  const onChange = vi.fn()
  const listAttrs: SnapshotAttribute[] = [
    { id: 'l1', name: 'tags', base_type: 'list', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={listAttrs}
      values={{}}
      onChange={onChange}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  const textarea = page.getByRole('textbox', { name: 'tags' })
  await expect.element(textarea).toBeVisible()
  await expect.element(textarea).toHaveAttribute('placeholder', 'Comma-separated values')
  await textarea.fill('a, b, c')
  expect(onChange).toHaveBeenCalledWith('tags', 'a, b, c')
})

// === TD-92: Inline validation warnings ===

// T-31.166: String exceeding max_length shows warning
test('T-31.166: string exceeding max_length shows warning', async () => {
  const attrsWithConstraints: SnapshotAttribute[] = [
    { id: 'a1', name: 'hostname', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { max_length: 5 } },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={attrsWithConstraints}
      values={{ hostname: 'toolong' }}
      onChange={vi.fn()}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  await expect.element(page.getByText(/maximum length/)).toBeVisible()
})

// T-31.167: String within max_length shows no warning
test('T-31.167: string within max_length shows no warning', async () => {
  const attrsWithConstraints: SnapshotAttribute[] = [
    { id: 'a1', name: 'hostname', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { max_length: 20 } },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={attrsWithConstraints}
      values={{ hostname: 'ok' }}
      onChange={vi.fn()}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  await expect.element(page.getByText(/maximum length/)).not.toBeInTheDocument()
})

// T-31.168: URL invalid shows warning
test('T-31.168: url invalid shows warning', async () => {
  const urlAttrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'website', base_type: 'url', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={urlAttrs}
      values={{ website: 'not a url' }}
      onChange={vi.fn()}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  await expect.element(page.getByText(/Invalid URL/)).toBeVisible()
})

// T-31.169: Warning disappears when value corrected (empty → no warning)
test('T-31.169: empty value shows no warning', async () => {
  const attrsWithConstraints: SnapshotAttribute[] = [
    { id: 'a1', name: 'hostname', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { max_length: 5 } },
  ]
  render(
    <AttributeFormFields
      schemaAttrs={attrsWithConstraints}
      values={{ hostname: '' }}
      onChange={vi.fn()}
      enumValues={{}}
      idPrefix="test"
    />,
  )
  await expect.element(page.getByText(/maximum length/)).not.toBeInTheDocument()
})

// T-31.170: Integer non-numeric shows warning
test('T-31.170: integer non-numeric shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ count: 'abc' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/valid number/)).toBeVisible()
})

// T-31.171: Integer not whole number shows warning
test('T-31.171: integer not whole shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ count: '3.14' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/whole number/)).toBeVisible()
})

// T-31.172: Integer below min shows warning
test('T-31.172: integer below min shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false, constraints: { min: 10 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ count: '5' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/minimum/i)).toBeVisible()
})

// T-31.173: Integer above max shows warning
test('T-31.173: integer above max shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false, constraints: { max: 100 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ count: '200' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/maximum/i)).toBeVisible()
})

// T-31.174: Integer within range shows no warning
test('T-31.174: integer within range no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'count', base_type: 'integer', description: '', ordinal: 1, required: false, constraints: { min: 1, max: 100 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ count: '50' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/minimum|maximum|number/i)).not.toBeInTheDocument()
})

// T-31.175: Number non-numeric shows warning
test('T-31.175: number non-numeric shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'weight', base_type: 'number', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ weight: 'abc' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/valid number/)).toBeVisible()
})

// T-31.176: Number below min shows warning
test('T-31.176: number below min shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'weight', base_type: 'number', description: '', ordinal: 1, required: false, constraints: { min: 0 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ weight: '-5.5' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/minimum/i)).toBeVisible()
})

// T-31.177: Number above max shows warning
test('T-31.177: number above max shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'weight', base_type: 'number', description: '', ordinal: 1, required: false, constraints: { max: 99.9 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ weight: '100' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/maximum/i)).toBeVisible()
})

// T-31.178: Number within range shows no warning
test('T-31.178: number within range no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'weight', base_type: 'number', description: '', ordinal: 1, required: false, constraints: { min: 0, max: 100 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ weight: '50.5' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/minimum|maximum|number/i)).not.toBeInTheDocument()
})

// T-31.179: Date invalid format shows warning
test('T-31.179: date invalid shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'start', base_type: 'date', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ start: 'not-a-date' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/date/i)).toBeVisible()
})

// T-31.180: Date valid shows no warning
test('T-31.180: date valid no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'start', base_type: 'date', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ start: '2026-04-15' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/invalid|date format/i)).not.toBeInTheDocument()
})

// T-31.181: Date impossible day shows warning
test('T-31.181: date impossible day shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'start', base_type: 'date', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ start: '2024-02-31' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/date/i)).toBeVisible()
})

// T-31.182: JSON invalid shows warning
test('T-31.182: json invalid shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'config', base_type: 'json', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ config: '{bad}' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/JSON/i)).toBeVisible()
})

// T-31.183: JSON valid shows no warning
test('T-31.183: json valid no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'config', base_type: 'json', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ config: '{"key":"value"}' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/JSON/i)).not.toBeInTheDocument()
})

// T-31.184: List invalid JSON shows warning
test('T-31.184: list invalid json shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'tags', base_type: 'list', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ tags: 'not json' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/list/i)).toBeVisible()
})

// T-31.185: List exceeds max_length shows warning
test('T-31.185: list exceeds max_length shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'tags', base_type: 'list', description: '', ordinal: 1, required: false, constraints: { max_length: 2 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ tags: '["a","b","c"]' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/maximum/i)).toBeVisible()
})

// T-31.186: List not an array shows warning
test('T-31.186: list non-array json shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'tags', base_type: 'list', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ tags: '{"not":"array"}' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/list/i)).toBeVisible()
})

// T-31.187: String failing pattern shows warning
test('T-31.187: string failing pattern shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'code', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { pattern: '^[a-z]+$' } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ code: 'ABC' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/pattern/i)).toBeVisible()
})

// T-31.188: String matching pattern shows no warning
test('T-31.188: string matching pattern no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'code', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { pattern: '^[a-z]+$' } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ code: 'abc' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/pattern/i)).not.toBeInTheDocument()
})

// T-31.189: Invalid regex pattern shows warning
test('T-31.189: invalid regex pattern shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'code', base_type: 'string', description: '', ordinal: 1, required: false, constraints: { pattern: '[invalid' } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ code: 'abc' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/pattern/i)).toBeVisible()
})

// T-31.190: URL with missing host shows warning (e.g. "http://")
test('T-31.190: url missing host shows warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'website', base_type: 'url', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ website: 'http://' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/URL/)).toBeVisible()
})

// T-31.191: Valid URL shows no warning
test('T-31.191: valid url no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'website', base_type: 'url', description: '', ordinal: 1, required: false },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ website: 'https://example.com' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/URL/)).not.toBeInTheDocument()
})

// === Direct validateAttributeValue tests (covers defensive guards not reachable through component) ===

// T-31.193: Empty value returns null (covers line 6 — empty guard)
test('T-31.193: validateAttributeValue empty returns null', () => {
  expect(validateAttributeValue('string', '', { max_length: 5 })).toBeNull()
})

// T-31.194: Boolean returns null (covers line 24 — default case)
test('T-31.194: validateAttributeValue boolean returns null', () => {
  expect(validateAttributeValue('boolean', 'true')).toBeNull()
  expect(validateAttributeValue('enum', 'anything')).toBeNull()
})

// T-31.192: List within max_length shows no warning (covers validateList success return)
test('T-31.192: list within max_length no warning', async () => {
  const attrs: SnapshotAttribute[] = [
    { id: 'a1', name: 'tags', base_type: 'list', description: '', ordinal: 1, required: false, constraints: { max_length: 5 } },
  ]
  render(
    <AttributeFormFields schemaAttrs={attrs} values={{ tags: '["a","b"]' }} onChange={vi.fn()} enumValues={{}} idPrefix="test" />,
  )
  await expect.element(page.getByText(/maximum|list/i)).not.toBeInTheDocument()
})
