import { describe, test, expect } from 'vitest'
import { errorMessage } from './errorMessage'

describe('errorMessage', () => {
  test('returns Error.message for Error instances', () => {
    expect(errorMessage(new Error('something broke'))).toBe('something broke')
  })

  test('returns default message for non-Error values', () => {
    expect(errorMessage('string error', 'fallback')).toBe('fallback')
    expect(errorMessage(42, 'fallback')).toBe('fallback')
    expect(errorMessage(null, 'fallback')).toBe('fallback')
    expect(errorMessage(undefined, 'fallback')).toBe('fallback')
    expect(errorMessage({}, 'fallback')).toBe('fallback')
  })

  test('uses generic default when no fallback provided', () => {
    expect(errorMessage('oops')).toBe('An error occurred')
  })
})
