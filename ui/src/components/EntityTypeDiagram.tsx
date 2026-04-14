import { useEffect, useState } from 'react'
import {
  observer,
  Visualization,
  VisualizationProvider,
  VisualizationSurface,
  ModelKind,
  GraphComponent,
  DefaultNode,
  NodeShape,
  EdgeStyle,
  EdgeTerminalType,
  EdgeConnectorArrow,
  TopologyControlBar,
  createTopologyControlButtons,
  defaultControlButtonsOptions,
  action,
  useVisualizationController,
  DagreLayout,
  withPanZoom,
  withSelection,
  TopologyView,
  GRAPH_LAYOUT_END_EVENT,
  useEventListener,
} from '@patternfly/react-topology'
import type {
  Model,
  NodeModel,
  EdgeModel,
  ComponentFactory,
  Graph,
  Layout,
  LayoutFactory,
  GraphElement,
  Edge,
  Node,
  Point,
} from '@patternfly/react-topology'
import type { SnapshotAttribute, SnapshotAssociation, EntityType } from '../types'

export interface DiagramEntityType {
  entityType: EntityType
  version: number
  attributes: SnapshotAttribute[]
  associations: SnapshotAssociation[]
}

export interface EdgeClickData {
  name: string
  assocType: string
  sourceRole: string
  targetRole: string
  sourceCardinality: string
  targetCardinality: string
  sourceEntityTypeId: string
  sourceEntityTypeName: string
  targetEntityTypeName: string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function buildEdgeClickData(data: Record<string, any>): EdgeClickData {
  return {
    name: data.name,
    assocType: data.assocType,
    sourceRole: data.sourceRole || '',
    targetRole: data.targetRole || '',
    sourceCardinality: data.sourceCardinality || '',
    targetCardinality: data.targetCardinality || '',
    sourceEntityTypeId: data.sourceEntityTypeId,
    sourceEntityTypeName: data.sourceEntityTypeName || '',
    targetEntityTypeName: data.targetEntityTypeName || '',
  }
}

interface EntityTypeDiagramProps {
  entityTypes: DiagramEntityType[]
  onNodeDoubleClick?: (entityTypeId: string) => void
  onEdgeClick?: (edgeData: EdgeClickData) => void
}

// ─── Node Component ──────────────────────────────────────────────────

const EntityTypeNode: React.FunctionComponent<{
  element: GraphElement
  selected?: boolean
  onSelect?: () => void
}> = observer(({ element, ...rest }: { element: GraphElement; [key: string]: any }) => {
  const node = element as Node
  const data = node.getData() || {}
  const attrs: SnapshotAttribute[] = data.attributes || []
  const version = data.version || 1
  const { width } = node.getDimensions()
  const headerHeight = 26

  return (
    <g onDoubleClick={() => data.onDoubleClick?.(node.getId())}>
      <DefaultNode element={element} {...rest} showLabel={false}>
        <rect x={0} y={0} width={width} height={headerHeight} fill="#f0f0f0" rx={0} ry={0} />
        <rect x={0} y={headerHeight - 6} width={width} height={6} fill="#f0f0f0" />
        <line x1={0} y1={headerHeight} x2={width} y2={headerHeight} stroke="#d2d2d2" strokeWidth={1} />
        <text x={width / 2} y={headerHeight / 2 + 1} textAnchor="middle" dominantBaseline="central"
          fontSize={12} fontWeight="bold" fill="#151515">
          {node.getLabel()} (V{version})
        </text>
        {attrs.map((attr, i) => {
          const typeName = attr.type_name || attr.base_type || 'unknown'
          return (
            <text key={attr.id || i} x={8} y={headerHeight + 14 + i * 14}
              fontSize={10} fill="#6a6e73" fontFamily="var(--pf-t--global--font--family--mono, monospace)">
              {attr.required ? '* ' : ''}{attr.name} : {typeName}
            </text>
          )
        })}
      </DefaultNode>
    </g>
  )
})

// ─── Edge Component ──────────────────────────────────────────────────

const AssociationEdge: React.FunctionComponent<{
  element: GraphElement
}> = observer(({ element }: { element: GraphElement }) => {
  const edge = element as Edge
  const data = edge.getData() || {}
  const startPoint = edge.getStartPoint()
  const endPoint = edge.getEndPoint()
  const bendpoints = edge.getBendpoints()

  const d = `M${startPoint.x} ${startPoint.y} ${bendpoints.map((b: Point) => `L${b.x} ${b.y} `).join('')}L${endPoint.x} ${endPoint.y}`

  const isContainment = data.assocType === 'containment'
  const isBidirectional = data.assocType === 'bidirectional'
  const strokeColor = isContainment ? '#3e8635' : isBidirectional ? '#6753ac' : '#6a6e73'
  const strokeDash = isContainment ? undefined : isBidirectional ? '8,4' : '5,3'

  const srcCard = data.sourceCardinality || '?'
  const tgtCard = data.targetCardinality || '?'
  const labelText = data.name ? `${data.name} [${srcCard} → ${tgtCard}]` : ''

  const t = 0.4
  const labelPoint = bendpoints.length > 0
    ? bendpoints[Math.floor(bendpoints.length * t)]
    : { x: startPoint.x + (endPoint.x - startPoint.x) * t, y: startPoint.y + (endPoint.y - startPoint.y) * t }

  const labelBg = isContainment ? '#f3faf2' : isBidirectional ? '#e7f1fa' : '#f5f5f5'
  const labelBorder = isContainment ? '#3e8635' : isBidirectional ? '#6753ac' : '#d2d2d2'
  const labelWidth = Math.max(labelText.length * 6.5 + 12, 60)
  const labelHeight = 18

  return (
    <>
      <path d={d} fill="none" stroke="transparent" strokeWidth={12}
        style={{ cursor: data.onEdgeClick ? 'pointer' : 'default' }}
        onClick={() => data.onEdgeClick?.(buildEdgeClickData(data))} />
      <path d={d} fill="none" stroke={strokeColor} strokeWidth={1.5} strokeDasharray={strokeDash}
        markerEnd={isBidirectional ? `url(#arrow-filled-${edge.getId()})` : undefined}
        markerStart={isBidirectional ? `url(#arrow-hollow-${edge.getId()})` : isContainment ? `url(#diamond-${edge.getId()})` : undefined} />
      {/* Diamond marker for containment edges (UML composition notation) */}
      {isContainment && (
        <g data-testid="diamond-source">
          <defs>
            <marker id={`diamond-${edge.getId()}`} viewBox="0 0 12 8"
              refX="0" refY="4" markerWidth={12} markerHeight={8} orient="auto">
              <path d="M 0 4 L 6 0 L 12 4 L 6 8 Z" fill={strokeColor} />
            </marker>
          </defs>
        </g>
      )}
      {/* Arrowhead markers for bidirectional edges */}
      {isBidirectional && (
        <g data-testid="hollow-arrow-source">
          <defs>
            {/* Filled arrowhead at target end */}
            <marker id={`arrow-filled-${edge.getId()}`} viewBox="0 0 10 10"
              refX="10" refY="5" markerWidth={10} markerHeight={10} orient="auto">
              <path d="M 0 0 L 10 5 L 0 10 Z" fill={strokeColor} />
            </marker>
            {/* Hollow arrowhead at source end */}
            <marker id={`arrow-hollow-${edge.getId()}`} viewBox="0 0 10 10"
              refX="0" refY="5" markerWidth={10} markerHeight={10} orient="auto">
              <path d="M 10 0 L 0 5 L 10 10 Z" fill="white" stroke={strokeColor} strokeWidth={1.5} />
            </marker>
          </defs>
        </g>
      )}
      {/* Filled arrowhead for non-bidirectional, non-containment edges */}
      {!isBidirectional && !isContainment && <EdgeConnectorArrow edge={edge} terminalType={EdgeTerminalType.directional} />}
      {/* Arrowhead at target end for containment edges */}
      {isContainment && <EdgeConnectorArrow edge={edge} terminalType={EdgeTerminalType.directional} />}
      {labelText && (
        <g transform={`translate(${labelPoint.x - labelWidth / 2}, ${labelPoint.y - labelHeight / 2})`}
          style={{ cursor: data.onEdgeClick ? 'pointer' : 'default' }}
          onClick={() => data.onEdgeClick?.(buildEdgeClickData(data))}>
          <rect width={labelWidth} height={labelHeight} rx={3} ry={3}
            fill={labelBg} stroke={labelBorder} strokeWidth={1} />
          <text x={labelWidth / 2} y={labelHeight / 2 + 1} textAnchor="middle" dominantBaseline="central"
            fontSize={10} fontWeight={500} fill="#151515">
            {labelText}
          </text>
        </g>
      )}
    </>
  )
})

// ─── Factories ──────────────────────────────────────────────────────

const layoutFactory: LayoutFactory = (_type: string, graph: Graph): Layout | undefined => {
  return new DagreLayout(graph, {
    nodeDistance: 60,
    rankdir: 'TB',
  })
}

const componentFactory: ComponentFactory = (kind: ModelKind, _type: string): any => {
  switch (kind) {
    case ModelKind.graph:
      return withPanZoom()(GraphComponent)
    case ModelKind.node:
      return withSelection()(EntityTypeNode as any)
    case ModelKind.edge:
      return AssociationEdge
    default:
      return undefined
  }
}

// ─── Model Builder ──────────────────────────────────────────────────

export function buildModel(
  entityTypes: DiagramEntityType[],
  onNodeDoubleClick?: (id: string) => void,
  onEdgeClick?: EntityTypeDiagramProps['onEdgeClick'],
): Model {
  const attrLineHeight = 14
  const headerPadding = 40

  const minNodeWidth = 200
  const charWidth = 7
  const nodePadding = 24

  const nodes: NodeModel[] = entityTypes.map((et) => {
    const attrHeight = Math.max(et.attributes.length * attrLineHeight + 8, 20)

    // Compute width dynamically from the longest attribute label
    let longestLabel = 0
    for (const attr of et.attributes) {
      const typeName = attr.type_name || attr.base_type || 'unknown'
      const prefix = attr.required ? '* ' : ''
      const label = `${prefix}${attr.name} : ${typeName}`
      longestLabel = Math.max(longestLabel, label.length)
    }
    const dynamicWidth = Math.max(minNodeWidth, longestLabel * charWidth + nodePadding)

    return {
      id: et.entityType.id,
      type: 'entity-type',
      label: et.entityType.name,
      width: dynamicWidth,
      height: headerPadding + attrHeight,
      shape: NodeShape.rect,
      data: {
        version: et.version,
        attributes: et.attributes,
        onDoubleClick: onNodeDoubleClick,
      },
    }
  })

  const edges: EdgeModel[] = []
  const entityIds = new Set(entityTypes.map((et) => et.entityType.id))
  for (const et of entityTypes) {
    for (const assoc of et.associations) {
      if (assoc.direction !== 'outgoing') continue
      if (!entityIds.has(assoc.target_entity_type_id)) continue
      edges.push({
        id: `edge-${et.entityType.id}-${assoc.id}`,
        type: 'association',
        source: et.entityType.id,
        target: assoc.target_entity_type_id,
        edgeStyle: assoc.type === 'containment' ? EdgeStyle.solid
          : assoc.type === 'bidirectional' ? EdgeStyle.dashedMd
          : EdgeStyle.dashed,
        data: {
          name: assoc.name,
          assocType: assoc.type,
          sourceRole: assoc.source_role,
          targetRole: assoc.target_role,
          sourceCardinality: assoc.source_cardinality,
          targetCardinality: assoc.target_cardinality,
          sourceEntityTypeId: et.entityType.id,
          sourceEntityTypeName: et.entityType.name,
          targetEntityTypeName: entityTypes.find(e => e.entityType.id === assoc.target_entity_type_id)?.entityType.name || assoc.target_entity_type_id,
          onEdgeClick,
        },
      })
    }
  }

  return {
    graph: {
      id: 'entity-type-graph',
      type: 'graph',
      layout: 'Cola',
    },
    nodes,
    edges,
  }
}

// ─── Diagram Content (follows official PF demo pattern) ─────────────

function DiagramContent({ entityTypes, onNodeDoubleClick, onEdgeClick }: EntityTypeDiagramProps) {
  const controller = useVisualizationController()
  const hasGraph = controller.hasGraph()

  // Phase 1: Set the model when data changes
  useEffect(() => {
    if (entityTypes.length === 0) return
    const model = buildModel(entityTypes, onNodeDoubleClick, onEdgeClick)
    controller.fromModel(model, true)
    controller.getGraph().layout()
  }, [controller, entityTypes, onNodeDoubleClick, onEdgeClick])

  // Phase 2: Once graph exists (surface has dimensions), run layout
  useEffect(() => {
    if (hasGraph) {
      controller.getGraph().layout()
    }
  }, [hasGraph, controller])

  // Fit to screen after layout completes
  useEventListener(GRAPH_LAYOUT_END_EVENT, () => {
    controller.getGraph().fit(60)
  })

  return (
    <TopologyView
      controlBar={
        <TopologyControlBar
          controlButtons={createTopologyControlButtons({
            ...defaultControlButtonsOptions,
            zoomInCallback: action(() => controller.getGraph().scaleBy(4 / 3)),
            zoomOutCallback: action(() => controller.getGraph().scaleBy(3 / 4)),
            fitToScreenCallback: action(() => controller.getGraph().fit(60)),
            resetViewCallback: action(() => controller.getGraph().reset()),
            legend: false,
          })}
        />
      }
    >
      <VisualizationSurface />
    </TopologyView>
  )
}

// ─── Main Component ─────────────────────────────────────────────────

export default function EntityTypeDiagram(props: EntityTypeDiagramProps) {
  const [controller] = useState(() => {
    const c = new Visualization()
    c.registerLayoutFactory(layoutFactory)
    c.registerComponentFactory(componentFactory)
    return c
  })

  // Cleanup on unmount — destroy graph to release SVG DOM refs and event listeners
  useEffect(() => {
    return () => {
      try {
        if (controller.hasGraph()) {
          controller.getGraph().destroy()
        }
      } catch {
        // ignore cleanup errors
      }
    }
  }, [controller])

  return (
    <div
      data-testid="entity-type-diagram"
      className="entity-type-diagram"
      style={{ height: 'calc(100vh - 180px)', minHeight: '400px' }}
    >
      <VisualizationProvider controller={controller}>
        <DiagramContent {...props} />
      </VisualizationProvider>
    </div>
  )
}
