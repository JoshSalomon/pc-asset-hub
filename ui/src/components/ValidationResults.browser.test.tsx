import { render } from 'vitest-browser-react'
import { expect, test } from 'vitest'
import { page } from 'vitest/browser'
import ValidationResults from './ValidationResults'
import type { ValidationError } from '../types'

test('many errors render within a scrollable container', async () => {
  const errors: ValidationError[] = Array.from({ length: 50 }, (_, i) => ({
    entity_type: 'Server',
    instance_name: `server-${i}`,
    field: 'hostname',
    violation: `required attribute "hostname" is missing a value`,
  }))
  render(<ValidationResults errors={errors} ran={true} />)

  await expect.element(page.getByRole('listitem').first()).toBeVisible()

  // Verify the scrollable container exists
  const scrollContainer = page.getByTestId('validation-error-list')
  await expect.element(scrollContainer).toBeVisible()
})

test('success alert shown when no errors', async () => {
  render(<ValidationResults errors={[]} ran={true} />)
  await expect.element(page.getByText('Validation passed')).toBeVisible()
})

test('nothing shown when not ran', async () => {
  render(<ValidationResults errors={[]} ran={false} />)
  // When ran=false and no error, component returns null — no alert should be visible
  await expect.element(page.getByRole('alert')).not.toBeInTheDocument()
})

test('error alert shown for request failure', async () => {
  render(<ValidationResults errors={[]} ran={false} error="network failure" />)
  await expect.element(page.getByText(/network failure/)).toBeVisible()
})
