import { describe, test, expect } from 'vitest'
import { buildTypedAttrs } from './buildTypedAttrs'
import type { SnapshotAttribute } from '../types'

const makeAttr = (name: string, type: string): SnapshotAttribute => ({
  id: `attr-${name}`, name, type, description: '', ordinal: 0, required: false,
})

describe('buildTypedAttrs (browser)', () => {
  test('skips empty string values', () => {
    const result = buildTypedAttrs({ hostname: '', port: '80' }, [makeAttr('hostname', 'string'), makeAttr('port', 'number')])
    expect(result).toEqual({ port: 80 })
  })
})
