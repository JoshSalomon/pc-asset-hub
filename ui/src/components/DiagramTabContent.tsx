import {
  Alert,
  EmptyState,
  EmptyStateBody,
  Spinner,
} from '@patternfly/react-core'
import EntityTypeDiagram from './EntityTypeDiagram'
import type { DiagramEntityType, EdgeClickData } from './EntityTypeDiagram'

interface DiagramTabContentProps {
  diagramData: DiagramEntityType[]
  diagramLoading: boolean
  diagramError: string | null
  onEdgeClick?: (edgeData: EdgeClickData) => void
}

export default function DiagramTabContent({
  diagramData,
  diagramLoading,
  diagramError,
  onEdgeClick,
}: DiagramTabContentProps) {
  if (diagramLoading) {
    return <Spinner aria-label="Loading diagram" />
  }

  return (
    <>
      {diagramError && (
        <Alert variant="danger" title={diagramError} isInline style={{ marginBottom: '1rem' }} />
      )}
      {diagramData.length === 0 && !diagramError ? (
        <EmptyState>
          <EmptyStateBody>
            No model diagram available. The catalog version has no pinned entity types.
          </EmptyStateBody>
        </EmptyState>
      ) : diagramData.length > 0 ? (
        <EntityTypeDiagram entityTypes={diagramData} {...(onEdgeClick ? { onEdgeClick } : {})} />
      ) : null}
    </>
  )
}
