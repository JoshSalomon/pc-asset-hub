import { describe, test, expect } from 'vitest'
import { buildTypedAttrs } from './buildTypedAttrs'
import type { SnapshotAttribute } from '../types'

const makeAttr = (name: string, base_type: string): SnapshotAttribute => ({
  id: `attr-${name}`, name, base_type, description: '', ordinal: 0, required: false,
})

// T-20.01: Converts string value to parseFloat for number-type attr
describe('buildTypedAttrs', () => {
  test('T-20.01: converts string to parseFloat for number type', () => {
    const result = buildTypedAttrs({ weight: '3.14' }, [makeAttr('weight', 'number')])
    expect(result).toEqual({ weight: 3.14 })
  })

  // T-20.02: Passes through string value for string-type attr
  test('T-20.02: passes through string value for string type', () => {
    const result = buildTypedAttrs({ hostname: 'foo' }, [makeAttr('hostname', 'string')])
    expect(result).toEqual({ hostname: 'foo' })
  })

  // T-20.03: Passes through string value for enum-type attr
  test('T-20.03: passes through string value for enum type', () => {
    const result = buildTypedAttrs({ status: 'active' }, [makeAttr('status', 'enum')])
    expect(result).toEqual({ status: 'active' })
  })

  // T-20.04: Skips empty string values
  test('T-20.04: skips empty string values', () => {
    const result = buildTypedAttrs({ hostname: '' }, [makeAttr('hostname', 'string')])
    expect(result).toEqual({})
  })

  // T-20.05: Returns empty object for empty input
  test('T-20.05: returns empty object for empty input', () => {
    const result = buildTypedAttrs({}, [])
    expect(result).toEqual({})
  })

  // T-20.06: Handles mix of types correctly
  test('T-20.06: handles mix of types correctly', () => {
    const attrs = [
      makeAttr('port', 'number'),
      makeAttr('hostname', 'string'),
      makeAttr('empty', 'string'),
    ]
    const result = buildTypedAttrs({ port: '8080', hostname: 'srv-1', empty: '' }, attrs)
    expect(result).toEqual({ port: 8080, hostname: 'srv-1' })
  })

  test('T-20.07: converts integer type to parseInt', () => {
    const result = buildTypedAttrs({ count: '42' }, [makeAttr('count', 'integer')])
    expect(result).toEqual({ count: 42 })
  })

  // TD-98: on edit, empty string sends null to clear the value
  test('T-20.08: edit mode sends null for cleared string field', () => {
    const result = buildTypedAttrs({ hostname: '' }, [makeAttr('hostname', 'string')], true)
    expect(result).toEqual({ hostname: null })
  })

  test('T-20.09: edit mode sends null for cleared number field', () => {
    const result = buildTypedAttrs({ port: '' }, [makeAttr('port', 'number')], true)
    expect(result).toEqual({ port: null })
  })

  test('T-20.10: edit mode sends null for cleared integer field', () => {
    const result = buildTypedAttrs({ count: '' }, [makeAttr('count', 'integer')], true)
    expect(result).toEqual({ count: null })
  })

  test('T-20.11: edit mode keeps non-empty values unchanged', () => {
    const result = buildTypedAttrs({ hostname: 'server-1' }, [makeAttr('hostname', 'string')], true)
    expect(result).toEqual({ hostname: 'server-1' })
  })

  test('T-20.12: create mode still skips empty values', () => {
    const result = buildTypedAttrs({ hostname: '' }, [makeAttr('hostname', 'string')], false)
    expect(result).toEqual({})
  })
})
