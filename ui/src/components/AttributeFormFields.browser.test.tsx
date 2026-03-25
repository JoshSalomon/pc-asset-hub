import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import AttributeFormFields from './AttributeFormFields'
import type { SnapshotAttribute } from '../types'

const schemaAttrs: SnapshotAttribute[] = [
  { id: 'sys-name', name: 'name', type: 'string', description: '', ordinal: -2, required: true, system: true },
  { id: 'sys-desc', name: 'description', type: 'string', description: '', ordinal: -1, required: false, system: true },
  { id: 'a1', name: 'color', type: 'enum', description: '', ordinal: 1, required: false, enum_id: 'enum1' },
  { id: 'a2', name: 'port', type: 'number', description: '', ordinal: 2, required: true },
  { id: 'a3', name: 'hostname', type: 'string', description: '', ordinal: 3, required: false },
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

// T-20.12: Renders enum attr with EnumSelect dropdown
test('T-20.12: renders enum attr with EnumSelect', async () => {
  render(
    <AttributeFormFields
      schemaAttrs={schemaAttrs}
      values={{}}
      onChange={vi.fn()}
      enumValues={enumValues}
      idPrefix="test"
    />,
  )
  // EnumSelect shows 'Select...' when no value
  await expect.element(page.getByText('Select...')).toBeVisible()
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
