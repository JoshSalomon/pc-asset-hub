import { describe, test, expect } from 'vitest'
import { validateAttributeValue } from './validateAttributeValue'

describe('validateAttributeValue (browser coverage)', () => {
  test('url with file:// scheme (no host) returns warning', () => {
    const result = validateAttributeValue('url', 'file:///tmp')
    expect(result).toContain('URL')
    expect(result).toContain('scheme and host')
  })

  test('date validation works in browser context', () => {
    expect(validateAttributeValue('date', '2024-02-29')).toBeNull() // leap year
    expect(validateAttributeValue('date', '2023-02-29')).toContain('does not exist') // non-leap
  })

  test('list without constraints returns null for valid array', () => {
    // Exercises the falsy-constraints branch at L137
    expect(validateAttributeValue('list', '[1, 2, 3]')).toBeNull()
  })
})
