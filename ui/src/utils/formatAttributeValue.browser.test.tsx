import { render } from 'vitest-browser-react'
import { expect, test } from 'vitest'
import { page } from 'vitest/browser'
import { formatAttributeValue } from './formatAttributeValue'

function Wrapper({ type, value }: { type: string; value: string | number | null }) {
  return <div data-testid="output">{formatAttributeValue(type, value)}</div>
}

// T-31.135: URL renders as clickable link
test('url renders as clickable link', async () => {
  render(<Wrapper type="url" value="https://example.com/path" />)
  const link = page.getByRole('link')
  await expect.element(link).toBeVisible()
  await expect.element(link).toHaveTextContent('https://example.com/path')
  await expect.element(link).toHaveAttribute('href', 'https://example.com/path')
  await expect.element(link).toHaveAttribute('target', '_blank')
})

// T-31.136: Boolean "true" → "Yes", "false" → "No"
test('boolean true renders as Yes', async () => {
  render(<Wrapper type="boolean" value="true" />)
  await expect.element(page.getByText('Yes')).toBeVisible()
})

test('boolean false renders as No', async () => {
  render(<Wrapper type="boolean" value="false" />)
  await expect.element(page.getByText('No')).toBeVisible()
})

// T-31.137: Date renders formatted
test('date renders formatted', async () => {
  render(<Wrapper type="date" value="2026-04-15" />)
  const output = page.getByTestId('output')
  await expect.element(output).toBeVisible()
  // Should render some date text (locale-dependent)
  await expect.element(output).not.toHaveTextContent('')
})

// T-31.138: JSON renders in pre block
test('json renders in pre block', async () => {
  render(<Wrapper type="json" value='{"key":"value"}' />)
  // Should contain pretty-printed content with the key name
  await expect.element(page.getByText(/"key"/)).toBeVisible()
})

// T-31.139: List renders comma-separated
test('list renders comma-separated', async () => {
  render(<Wrapper type="list" value='["alpha","beta","gamma"]' />)
  await expect.element(page.getByText('alpha, beta, gamma')).toBeVisible()
})

// Null renders em-dash
test('null renders em dash', async () => {
  render(<Wrapper type="string" value={null} />)
  await expect.element(page.getByText('\u2014')).toBeVisible()
})

// Default type renders as String(value)
test('string renders as plain text', async () => {
  render(<Wrapper type="string" value="hello world" />)
  await expect.element(page.getByText('hello world')).toBeVisible()
})

// Number renders as string
test('number renders as string', async () => {
  render(<Wrapper type="number" value={42.5} />)
  await expect.element(page.getByText('42.5')).toBeVisible()
})

// Invalid JSON in list falls back to raw
test('list invalid json renders raw', async () => {
  render(<Wrapper type="list" value="not json" />)
  await expect.element(page.getByText('not json')).toBeVisible()
})

// Invalid JSON in json type falls back to raw
test('json invalid syntax renders raw', async () => {
  render(<Wrapper type="json" value="{bad}" />)
  await expect.element(page.getByText('{bad}')).toBeVisible()
})

// XSS: javascript: URL must not render as an <a> tag at all
test('javascript url renders as plain text without anchor tag', async () => {
  render(<Wrapper type="url" value="javascript:alert(1)" />)
  const output = page.getByTestId('output')
  await expect.element(output).toBeVisible()
  // Must not contain an <a> tag — defense-in-depth, don't rely on React sanitization
  const el = await output.element()
  expect(el.querySelector('a')).toBeNull()
})

// data: URL must not render as an <a> tag
test('data url renders as plain text without anchor tag', async () => {
  render(<Wrapper type="url" value="data:text/html,<h1>hi</h1>" />)
  const output = page.getByTestId('output')
  await expect.element(output).toBeVisible()
  const el = await output.element()
  expect(el.querySelector('a')).toBeNull()
})

// http URL still renders as link
test('http url renders as link', async () => {
  render(<Wrapper type="url" value="http://example.com" />)
  const link = page.getByRole('link')
  await expect.element(link).toBeVisible()
  await expect.element(link).toHaveAttribute('href', 'http://example.com')
})

// List with valid JSON but not an array falls back to raw
test('list non-array json renders raw', async () => {
  render(<Wrapper type="list" value='{"key":"value"}' />)
  await expect.element(page.getByText('{"key":"value"}')).toBeVisible()
})

// TD-117 / T-28.01: Corrupted boolean shows raw value with warning
test('boolean corrupted value shows raw value with warning', async () => {
  render(<Wrapper type="boolean" value="yes" />)
  const output = page.getByTestId('output')
  // Should show the raw value, not "No"
  await expect.element(output).toHaveTextContent('yes')
  // Should have a warning icon (⚠)
  await expect.element(output).toHaveTextContent('⚠')
})

test('boolean numeric 1 shows raw value with warning', async () => {
  render(<Wrapper type="boolean" value="1" />)
  const output = page.getByTestId('output')
  await expect.element(output).toHaveTextContent('1')
  await expect.element(output).toHaveTextContent('⚠')
})

// Copilot review: boolean warning accessible to screen readers
test('boolean corrupted value has accessible warning text', async () => {
  render(<Wrapper type="boolean" value="maybe" />)
  const output = page.getByTestId('output')
  await expect.element(output).toHaveTextContent('maybe')
  const el = await output.element()
  const emojiSpan = el.querySelector('[aria-hidden="true"]')
  expect(emojiSpan).not.toBeNull()
  expect(emojiSpan!.textContent).toBe('⚠')
  expect(el.textContent).toContain('Warning: unexpected boolean value')
})
