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
  sessionStorage.clear()
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

// TD-105 / T-28.14: Validation results saved to sessionStorage after validate
test('T-28.14: validation results saved to sessionStorage', async () => {
  const errors = [{ entity_type: 'X', instance_name: 'y', field: 'f', violation: 'missing required' }]
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'invalid', errors })
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('status')).toHaveTextContent('invalid')

  // sessionStorage should have the results
  const stored = sessionStorage.getItem('validation:my-catalog')
  expect(stored).not.toBeNull()
  const parsed = JSON.parse(stored!)
  expect(parsed.errors).toHaveLength(1)
  expect(parsed.ran).toBe(true)
})

// TD-105 / T-28.16: Validation results rehydrated from sessionStorage on mount
test('T-28.16: validation results rehydrated from sessionStorage on mount', async () => {
  // Pre-populate sessionStorage
  sessionStorage.setItem('validation:hydrate-test', JSON.stringify({
    errors: [{ entity_type: 'Srv', instance_name: 's1', field: 'name', violation: 'required' }],
    ran: true,
  }))
  render(<TestComponent catalogName="hydrate-test" />)
  await expect.element(page.getByTestId('status')).toHaveTextContent('invalid')
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('1')
})

// Coverage: corrupt sessionStorage data is ignored (loadFromSession catch block)
test('corrupt sessionStorage data is ignored on mount', async () => {
  // Put invalid JSON in sessionStorage
  sessionStorage.setItem('validation:corrupt-test', 'not valid json{{{')
  render(<TestComponent catalogName="corrupt-test" />)
  // Should start in idle state (corrupt data ignored)
  await expect.element(page.getByTestId('status')).toHaveTextContent('idle')
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('0')
})

// Coverage: sessionStorage with wrong shape is ignored
test('sessionStorage with wrong shape is ignored on mount', async () => {
  // Valid JSON but wrong shape (missing 'ran' boolean)
  sessionStorage.setItem('validation:bad-shape', JSON.stringify({ errors: 'not-an-array', ran: 'not-bool' }))
  render(<TestComponent catalogName="bad-shape" />)
  await expect.element(page.getByTestId('status')).toHaveTextContent('idle')
})

// TD-105 / T-28.15: Re-validate clears old results and stores new
test('T-28.15: re-validate updates sessionStorage', async () => {
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'invalid', errors: [{ entity_type: 'A', instance_name: 'b', field: 'c', violation: 'd' }] })
  render(<TestComponent catalogName="my-catalog" />)
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('1')

  // Re-validate with 0 errors
  ;(api.catalogs.validate as Mock).mockResolvedValue({ status: 'valid', errors: [] })
  await page.getByRole('button', { name: 'Validate' }).click()
  await expect.element(page.getByTestId('status')).toHaveTextContent('valid')
  await expect.element(page.getByTestId('error-count')).toHaveTextContent('0')

  // SessionStorage should have the updated (empty) results
  const stored = sessionStorage.getItem('validation:my-catalog')
  expect(stored).not.toBeNull()
  const parsed = JSON.parse(stored!)
  expect(parsed.errors).toEqual([])
})
