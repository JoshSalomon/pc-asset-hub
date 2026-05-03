import { expect, test, describe, vi } from 'vitest'
import { buildModel } from './EntityTypeDiagram'
import type { DiagramEntityType } from './EntityTypeDiagram'

const baseEntityTypes: DiagramEntityType[] = [
  {
    entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
    version: 1,
    attributes: [{ id: 'a1', name: 'hostname', description: '', base_type: 'string', ordinal: 1, required: false }],
    associations: [
      {
        id: 'assoc1', name: 'tools', type: 'containment', direction: 'outgoing',
        target_entity_type_id: 'et2', target_entity_type_name: 'Tool',
        source_role: 'parent', target_role: 'child',
        source_cardinality: '1', target_cardinality: '0..n',
      },
    ],
  },
  {
    entityType: { id: 'et2', name: 'Tool', created_at: '', updated_at: '' },
    version: 1,
    attributes: [],
    associations: [
      {
        id: 'assoc1-in', name: 'tools', type: 'containment', direction: 'incoming',
        target_entity_type_id: 'et1', target_entity_type_name: 'Server',
        source_entity_type_id: 'et1', source_entity_type_name: 'Server',
        source_role: 'parent', target_role: 'child',
        source_cardinality: '1', target_cardinality: '0..n',
      },
    ],
  },
]

function makeEntityTypes(assocOverrides: Partial<DiagramEntityType['associations'][0]>): DiagramEntityType[] {
  return [
    {
      entityType: { id: 'et1', name: 'A', created_at: '', updated_at: '' }, version: 1, attributes: [],
      associations: [{
        id: 'assoc1', name: 'link', type: 'directional', direction: 'outgoing',
        target_entity_type_id: 'et2', target_entity_type_name: 'B',
        source_role: '', target_role: '',
        source_cardinality: '0..n', target_cardinality: '0..n',
        ...assocOverrides,
      }],
    },
    { entityType: { id: 'et2', name: 'B', created_at: '', updated_at: '' }, version: 1, attributes: [], associations: [] },
  ]
}

describe('buildModel node width', () => {
  test('node width accommodates long attribute labels', () => {
    const longAttrEntityTypes: DiagramEntityType[] = [
      {
        entityType: { id: 'et1', name: 'Agent', created_at: '', updated_at: '' },
        version: 1,
        attributes: [
          { id: 'a1', name: 'execution-modes', description: '', base_type: 'enum', type_name: 'guardrail-invocation', ordinal: 1, required: true },
          { id: 'a2', name: 'name', description: '', base_type: 'string', ordinal: 2, required: false },
        ],
        associations: [],
      },
    ]
    const model = buildModel(longAttrEntityTypes)
    const node = model.nodes![0]
    // The longest label is "* execution-modes : guardrail-invocation" (40 chars)
    // A fixed width of 200 would be too narrow; the dynamic width should be wider
    expect(node.width).toBeGreaterThan(200)
  })

  test('node width uses minimum when attributes are short', () => {
    const shortAttrEntityTypes: DiagramEntityType[] = [
      {
        entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
        version: 1,
        attributes: [
          { id: 'a1', name: 'name', description: '', base_type: 'string', ordinal: 1, required: false },
        ],
        associations: [],
      },
    ]
    const model = buildModel(shortAttrEntityTypes)
    const node = model.nodes![0]
    // Short labels should still get the minimum width
    expect(node.width).toBe(200)
  })

  test('node width accounts for required attribute prefix', () => {
    const requiredAttrEntityTypes: DiagramEntityType[] = [
      {
        entityType: { id: 'et1', name: 'Config', created_at: '', updated_at: '' },
        version: 1,
        attributes: [
          { id: 'a1', name: 'very-long-configuration-parameter-name', description: '', base_type: 'enum', type_name: 'extended-enumeration-type', ordinal: 1, required: true },
        ],
        associations: [],
      },
    ]
    const model = buildModel(requiredAttrEntityTypes)
    const node = model.nodes![0]
    // "* very-long-configuration-parameter-name : extended-enumeration-type" = 69 chars
    expect(node.width).toBeGreaterThan(200)
  })
})

describe('buildModel edge data', () => {
  test('containment edge has assocType containment', () => {
    const model = buildModel(baseEntityTypes)
    const containmentEdge = model.edges!.find(e => e.data.assocType === 'containment')
    expect(containmentEdge).toBeDefined()
    expect(containmentEdge!.data.assocType).toBe('containment')
  })

  test('directional edge has assocType directional', () => {
    const ets = makeEntityTypes({ type: 'directional' })
    const model = buildModel(ets)
    const edge = model.edges![0]
    expect(edge.data.assocType).toBe('directional')
  })

  test('bidirectional edge has assocType bidirectional', () => {
    const ets = makeEntityTypes({ type: 'bidirectional' })
    const model = buildModel(ets)
    const edge = model.edges![0]
    expect(edge.data.assocType).toBe('bidirectional')
  })

  test('onEdgeClick callback is passed through in edge data', () => {
    const handler = vi.fn()
    const model = buildModel(baseEntityTypes, undefined, handler)
    const edge = model.edges![0]
    expect(edge.data.onEdgeClick).toBe(handler)
  })

  test('edge data contains source and target entity type info', () => {
    const model = buildModel(baseEntityTypes)
    const edge = model.edges![0]
    expect(edge.data.sourceEntityTypeId).toBe('et1')
    expect(edge.data.sourceEntityTypeName).toBe('Server')
    expect(edge.data.targetEntityTypeName).toBe('Tool')
    expect(edge.data.sourceRole).toBe('parent')
    expect(edge.data.targetRole).toBe('child')
    expect(edge.data.sourceCardinality).toBe('1')
    expect(edge.data.targetCardinality).toBe('0..n')
  })
})

describe('buildModel edge filtering', () => {
  test('incoming association produces 0 edges', () => {
    const incomingOnly: DiagramEntityType[] = [
      {
        entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
        version: 1,
        attributes: [],
        associations: [
          {
            id: 'assoc-in', name: 'tools', type: 'containment', direction: 'incoming',
            target_entity_type_id: 'et2', target_entity_type_name: 'Tool',
            source_role: 'parent', target_role: 'child',
            source_cardinality: '1', target_cardinality: '0..n',
          },
        ],
      },
      {
        entityType: { id: 'et2', name: 'Tool', created_at: '', updated_at: '' },
        version: 1, attributes: [], associations: [],
      },
    ]
    const model = buildModel(incomingOnly)
    expect(model.edges!.length).toBe(0)
  })

  test('association to external (not-in-diagram) entity type produces 0 edges', () => {
    const externalTarget: DiagramEntityType[] = [
      {
        entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
        version: 1,
        attributes: [],
        associations: [
          {
            id: 'assoc-ext', name: 'external-link', type: 'directional', direction: 'outgoing',
            target_entity_type_id: 'et-external', target_entity_type_name: 'ExternalType',
            source_role: '', target_role: '',
            source_cardinality: '0..n', target_cardinality: '0..n',
          },
        ],
      },
    ]
    const model = buildModel(externalTarget)
    expect(model.edges!.length).toBe(0)
  })
})

// TD-83: Node dimensions must match rendered dimensions
describe('TD-83: node dimension consistency', () => {
  // The renderer uses headerHeight=26 and attributes at y = headerHeight + 14 + i*14
  // Model dimensions must match: height = 26 (header) + n*14 (attrs) + padding
  const RENDER_HEADER_HEIGHT = 26
  const ATTR_LINE_HEIGHT = 14

  test('T-28.20: single-attr node height matches renderer', () => {
    const model = buildModel(baseEntityTypes)
    const server = model.nodes!.find(n => n.id === 'et1')!
    // Server has 1 attribute: expected height = headerHeight + 1*14 + bottom padding
    const expectedMinHeight = RENDER_HEADER_HEIGHT + 1 * ATTR_LINE_HEIGHT
    expect(server.height).toBeGreaterThanOrEqual(expectedMinHeight)
    // Should not be significantly larger (old code had 14px excess)
    expect(server.height).toBeLessThanOrEqual(expectedMinHeight + 12)
  })

  test('T-28.21: zero-attr node height matches renderer', () => {
    const model = buildModel(baseEntityTypes)
    const tool = model.nodes!.find(n => n.id === 'et2')!
    // Tool has 0 attributes
    const expectedMinHeight = RENDER_HEADER_HEIGHT
    expect(tool.height).toBeGreaterThanOrEqual(expectedMinHeight)
    expect(tool.height).toBeLessThanOrEqual(expectedMinHeight + 20) // minimum box size
  })
})

