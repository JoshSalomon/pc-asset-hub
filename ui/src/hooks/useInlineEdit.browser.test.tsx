import { render } from 'vitest-browser-react'
import { expect, test, vi, beforeEach, type Mock } from 'vitest'
import { page } from 'vitest/browser'
import { api } from '../api/client'
import { useInlineEdit } from './useInlineEdit'

vi.mock('../api/client', () => ({
  api: {
    catalogVersions: { update: vi.fn() },
  },
  setAuthRole: vi.fn(),
}))

function TestComponent({ cvId, onSuccess, onError }: {
  cvId?: string
  onSuccess: () => void
  onError: (msg: string) => void
}) {
  const edit = useInlineEdit({ catalogVersionId: cvId, onSuccess, onError })
  return (
    <div>
      <span data-testid="editingLabel">{String(edit.editingLabel)}</span>
      <span data-testid="editLabelValue">{edit.editLabelValue}</span>
      <span data-testid="editingDesc">{String(edit.editingDesc)}</span>
      <span data-testid="editDescValue">{edit.editDescValue}</span>

      <button onClick={() => edit.startEditLabel('current-label')}>StartLabel</button>
      <button onClick={edit.cancelEditLabel}>CancelLabel</button>
      <button onClick={edit.handleSaveLabel}>SaveLabel</button>

      <button onClick={() => edit.startEditDesc('current-desc')}>StartDesc</button>
      <button onClick={edit.cancelEditDesc}>CancelDesc</button>
      <button onClick={edit.handleSaveDescription}>SaveDesc</button>

      <input
        data-testid="label-input"
        value={edit.editLabelValue}
        onChange={(e) => edit.setEditLabelValue(e.target.value)}
      />
      <input
        data-testid="desc-input"
        value={edit.editDescValue}
        onChange={(e) => edit.setEditDescValue(e.target.value)}
      />
    </div>
  )
}

let onSuccess: Mock
let onError: Mock

beforeEach(() => {
  vi.clearAllMocks()
  onSuccess = vi.fn()
  onError = vi.fn()
  ;(api.catalogVersions.update as Mock).mockResolvedValue({})
})

test('initial state: not editing', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await expect.element(page.getByTestId('editingLabel')).toHaveTextContent('false')
  await expect.element(page.getByTestId('editingDesc')).toHaveTextContent('false')
})

test('startEditLabel sets editing state and value', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartLabel' }).click()
  await expect.element(page.getByTestId('editingLabel')).toHaveTextContent('true')
  await expect.element(page.getByTestId('editLabelValue')).toHaveTextContent('current-label')
})

test('cancelEditLabel resets editing state', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartLabel' }).click()
  await expect.element(page.getByTestId('editingLabel')).toHaveTextContent('true')
  await page.getByRole('button', { name: 'CancelLabel' }).click()
  await expect.element(page.getByTestId('editingLabel')).toHaveTextContent('false')
})

test('handleSaveLabel calls API and onSuccess', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartLabel' }).click()
  await page.getByRole('button', { name: 'SaveLabel' }).click()
  expect(api.catalogVersions.update).toHaveBeenCalledWith('cv1', { version_label: 'current-label' })
  await vi.waitFor(() => expect(onSuccess).toHaveBeenCalled())
  await expect.element(page.getByTestId('editingLabel')).toHaveTextContent('false')
})

test('handleSaveLabel error calls onError', async () => {
  ;(api.catalogVersions.update as Mock).mockRejectedValue(new Error('409: conflict'))
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartLabel' }).click()
  await page.getByRole('button', { name: 'SaveLabel' }).click()
  await vi.waitFor(() => expect(onError).toHaveBeenCalledWith('409: conflict'))
})

test('handleSaveLabel does nothing without cvId', async () => {
  render(<TestComponent cvId={undefined} onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartLabel' }).click()
  await page.getByRole('button', { name: 'SaveLabel' }).click()
  expect(api.catalogVersions.update).not.toHaveBeenCalled()
})

test('startEditDesc sets editing state and value', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartDesc' }).click()
  await expect.element(page.getByTestId('editingDesc')).toHaveTextContent('true')
  await expect.element(page.getByTestId('editDescValue')).toHaveTextContent('current-desc')
})

test('cancelEditDesc resets editing state', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartDesc' }).click()
  await page.getByRole('button', { name: 'CancelDesc' }).click()
  await expect.element(page.getByTestId('editingDesc')).toHaveTextContent('false')
})

test('handleSaveDescription calls API and onSuccess', async () => {
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartDesc' }).click()
  await page.getByRole('button', { name: 'SaveDesc' }).click()
  expect(api.catalogVersions.update).toHaveBeenCalledWith('cv1', { description: 'current-desc' })
  await vi.waitFor(() => expect(onSuccess).toHaveBeenCalled())
  await expect.element(page.getByTestId('editingDesc')).toHaveTextContent('false')
})

test('handleSaveDescription error calls onError', async () => {
  ;(api.catalogVersions.update as Mock).mockRejectedValue(new Error('500: failed'))
  render(<TestComponent cvId="cv1" onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartDesc' }).click()
  await page.getByRole('button', { name: 'SaveDesc' }).click()
  await vi.waitFor(() => expect(onError).toHaveBeenCalledWith('500: failed'))
})

test('handleSaveDescription does nothing without cvId', async () => {
  render(<TestComponent cvId={undefined} onSuccess={onSuccess} onError={onError} />)
  await page.getByRole('button', { name: 'StartDesc' }).click()
  await page.getByRole('button', { name: 'SaveDesc' }).click()
  expect(api.catalogVersions.update).not.toHaveBeenCalled()
})
