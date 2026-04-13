import { describe, test, expect } from 'vitest'
import { buildTypedAttrs } from './buildTypedAttrs'
import type { SnapshotAttribute } from '../types'

const makeAttr = (name: string, base_type: string): SnapshotAttribute => ({
  id: `attr-${name}`, name, base_type, description: '', ordinal: 0, required: false,
})

describe('buildTypedAttrs (browser)', () => {
  test('skips empty string values', () => {
    const result = buildTypedAttrs({ hostname: '', port: '80' }, [makeAttr('hostname', 'string'), makeAttr('port', 'number')])
    expect(result).toEqual({ port: 80 })
  })

  test('parses integer base_type with parseInt', () => {
    const result = buildTypedAttrs({ count: '42' }, [makeAttr('count', 'integer')])
    expect(result).toEqual({ count: 42 })
    // Verify it's truly parseInt behavior (truncates decimals)
    const result2 = buildTypedAttrs({ count: '3.9' }, [makeAttr('count', 'integer')])
    expect(result2).toEqual({ count: 3 })
  })

  test('converts boolean base_type to true/false', () => {
    const schema = [makeAttr('active', 'boolean')]
    expect(buildTypedAttrs({ active: 'true' }, schema)).toEqual({ active: true })
    expect(buildTypedAttrs({ active: 'false' }, schema)).toEqual({ active: false })
    expect(buildTypedAttrs({ active: 'anything' }, schema)).toEqual({ active: false })
  })
})
