import { expect, test, describe } from 'vitest'
import { buildModel } from './EntityTypeDiagram'
import type { DiagramEntityType } from './EntityTypeDiagram'

const baseEntityTypes: DiagramEntityType[] = [
  {
    entityType: { id: 'et1', name: 'Server', created_at: '', updated_at: '' },
    version: 1,
    attributes: [{ id: 'a1', name: 'hostname', description: '', type: 'string', ordinal: 1, required: false }],
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
})
