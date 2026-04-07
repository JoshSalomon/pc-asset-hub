import { render } from 'vitest-browser-react'
import { expect, test, vi } from 'vitest'
import { page } from 'vitest/browser'
import DiagramTabContent from './DiagramTabContent'
import type { DiagramEntityType } from './EntityTypeDiagram'

// Mock EntityTypeDiagram to avoid topology rendering complexity
vi.mock('./EntityTypeDiagram', () => ({
  default: (props: { entityTypes: DiagramEntityType[]; onEdgeClick?: unknown }) => {
    const children = [`Diagram with ${props.entityTypes.length} types`]
    return (
      <div data-testid="entity-type-diagram" data-has-edge-click={props.onEdgeClick ? 'true' : 'false'}>
        {children}
      </div>
    )
  },
}))

const mockDiagramData: DiagramEntityType[] = [
  {
    entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
    version: 1,
    attributes: [],
    associations: [],
  },
]

test('shows spinner when loading', async () => {
  render(<DiagramTabContent diagramData={[]} diagramLoading={true} diagramError={null} />)
  await expect.element(page.getByRole('progressbar', { name: 'Loading diagram' })).toBeVisible()
})

test('shows error alert when error', async () => {
  render(<DiagramTabContent diagramData={[]} diagramLoading={false} diagramError="Network error" />)
  await expect.element(page.getByText('Network error')).toBeVisible()
})

test('shows empty state when no data and no error', async () => {
  render(<DiagramTabContent diagramData={[]} diagramLoading={false} diagramError={null} />)
  await expect.element(page.getByText(/No model diagram available/)).toBeVisible()
})

test('shows diagram when data present', async () => {
  render(<DiagramTabContent diagramData={mockDiagramData} diagramLoading={false} diagramError={null} />)
  await expect.element(page.getByTestId('entity-type-diagram')).toBeVisible()
  await expect.element(page.getByText('Diagram with 1 types')).toBeVisible()
})

test('passes onEdgeClick to EntityTypeDiagram when provided', async () => {
  const onEdgeClick = vi.fn()
  render(<DiagramTabContent diagramData={mockDiagramData} diagramLoading={false} diagramError={null} onEdgeClick={onEdgeClick} />)
  const diagram = page.getByTestId('entity-type-diagram')
  await expect.element(diagram).toBeVisible()
  expect(diagram.element().getAttribute('data-has-edge-click')).toBe('true')
})

test('does not pass onEdgeClick when not provided', async () => {
  render(<DiagramTabContent diagramData={mockDiagramData} diagramLoading={false} diagramError={null} />)
  const diagram = page.getByTestId('entity-type-diagram')
  await expect.element(diagram).toBeVisible()
  expect(diagram.element().getAttribute('data-has-edge-click')).toBe('false')
})

test('shows error alert AND empty state are not shown when loading', async () => {
  render(<DiagramTabContent diagramData={[]} diagramLoading={true} diagramError="stale error" />)
  // When loading, should show spinner, not error or empty state
  await expect.element(page.getByRole('progressbar', { name: 'Loading diagram' })).toBeVisible()
  expect(page.getByText(/No model diagram available/).elements().length).toBe(0)
})
