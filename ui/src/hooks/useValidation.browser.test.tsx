import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useValidation } from './useValidation'

vi.mock('../api/client', () => ({
  api: {
    catalogs: { validate: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

function TestComponent({ catalogName, onComplete }: { catalogName?: string; onComplete?: () => Promise<void> }) {
  const validation = useValidation(catalogName, onComplete)
  return (
    <div>
      <span data-testid="status">{validation.ran ? (validation.errors.length === 0 ? 'valid' : 'invalid') : 'idle'}</span>
      <span data-testid="validating">{String(validation.validating)}</span>
      <span data-testid="error-count">{validation.errors.length}</span>
      <span data-testid="error">{validation.error || ''}</span>
      <button onClick={validation.validate}>Validate</button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

// Cover: validate with undefined catalogName does nothing (early return)
test('useValidation: undefined catalogName does nothing on validate', async () => {
  render(<TestComponent catalogName={undefined} />)
  await page.getByRole('button', { name: 'Validate' }).click()
  expect(api.catalogs.validate).not.toHaveBeenCalled()
  await expect.element(page.getByTestId('status')).toHaveTextContent('idle')
})

// Cover: validate without onComplete callback
test('useValidation: works without onComplete callback', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'valid', errors: [] })
  render(<TestComponent catalogName="test-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  expect(api.catalogs.validate).toHaveBeenCalledWith('test-catalog')
  await expect.element(page.getByTestId('status')).toHaveTextContent('valid')
})

// Cover: validate with onComplete callback
test('useValidation: calls onComplete after validation', async () => {
  const onComplete = vi.fn().mockResolvedValue(undefined)
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'invalid', errors: [{ entity_type: 'X', instance_name: 'y', field: 'f', violation: 'v' }] })
  render(<TestComponent catalogName="test-catalog" onComplete={onComplete} />)
  await page.getByRole('button', { name: 'Validate' }).click()
  expect(onComplete).toHaveBeenCalled()
  await expect.element(page.getByTestId('status')).toHaveTextContent('invalid')
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('1')
})

// Cover: validate response with no errors field (null/undefined → [])
test('useValidation: handles missing errors field in response', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'valid' })
  render(<TestComponent catalogName="test-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('status')).toHaveTextContent('valid')
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('0')
})

// Cover: API throws non-Error value — fallback message used
test('useValidation: non-Error rejection uses fallback message', async () => {
  ;(api.catalogs.validate as Mock).mockRejectedValue('string error')
  render(<TestComponent catalogName="test-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('error')).toHaveTextContent('Validation failed')
})

// Cover: validate API throws error — error state is set, validating resets
test('useValidation: API error sets error state', async () => {
  ;(api.catalogs.validate as Mock).mockRejectedValue(new Error('network error'))
  render(<TestComponent catalogName="test-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('validating')).toHaveTextContent('false')
  await expect.element(page.getByTestId('status')).toHaveTextContent('idle')
  await expect.element(page.getByTestId('error')).toHaveTextContent('network error')
})
