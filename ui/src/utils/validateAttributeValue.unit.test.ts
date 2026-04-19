import { expect, test } from 'vitest'
import { validateAttributeValue } from './validateAttributeValue'

// Empty string always returns null (draft mode)
test('T-31.126a: empty string returns null', () => {
  expect(validateAttributeValue('string', '', { max_length: 5 })).toBeNull()
})

// T-31.126: String exceeding max_length → warning
test('T-31.126: string exceeding max_length', () => {
  const result = validateAttributeValue('string', 'toolong', { max_length: 5 })
  expect(result).toContain('maximum length')
})

// T-31.127: String within max_length → null
test('T-31.127: string within max_length', () => {
  expect(validateAttributeValue('string', 'ok', { max_length: 10 })).toBeNull()
})

// T-31.128: String failing pattern → warning
test('T-31.128: string failing pattern', () => {
  const result = validateAttributeValue('string', 'ABC', { pattern: '^[a-z]+$' })
  expect(result).toContain('pattern')
})

// T-31.129: String matching pattern → null
test('T-31.129: string matching pattern', () => {
  expect(validateAttributeValue('string', 'abc', { pattern: '^[a-z]+$' })).toBeNull()
})

// Invalid regex in pattern → warning about pattern
test('T-31.129a: invalid regex pattern', () => {
  const result = validateAttributeValue('string', 'abc', { pattern: '[invalid' })
  expect(result).toContain('pattern')
})

// Unanchored pattern: partial match must NOT pass
test('unanchored pattern rejects partial match', () => {
  const result = validateAttributeValue('string', 'ABCxyz', { pattern: '[0-9A-F]+' })
  expect(result).toContain('pattern')
})

test('unanchored pattern accepts full match', () => {
  expect(validateAttributeValue('string', 'ABCDEF', { pattern: '[0-9A-F]+' })).toBeNull()
})

// T-31.130: Integer not whole number → warning
test('T-31.130: integer not whole number', () => {
  const result = validateAttributeValue('integer', '3.14')
  expect(result).toContain('whole number')
})

// T-31.131: Integer below min → warning
test('T-31.131: integer below min', () => {
  const result = validateAttributeValue('integer', '5', { min: 10 })
  expect(result).toContain('minimum')
})

// Integer above max → warning
test('T-31.131a: integer above max', () => {
  const result = validateAttributeValue('integer', '200', { max: 100 })
  expect(result).toContain('maximum')
})

// T-31.132: Integer within range → null
test('T-31.132: integer within range', () => {
  expect(validateAttributeValue('integer', '50', { min: 1, max: 100 })).toBeNull()
})

// T-31.133: Number below min → warning
test('T-31.133: number below min', () => {
  const result = validateAttributeValue('number', '-5.5', { min: 0 })
  expect(result).toContain('minimum')
})

// Number above max → warning
test('T-31.133a: number above max', () => {
  const result = validateAttributeValue('number', '100', { max: 99.9 })
  expect(result).toContain('maximum')
})

// Number within range → null
test('T-31.133b: number within range', () => {
  expect(validateAttributeValue('number', '50.5', { min: 0, max: 100 })).toBeNull()
})

// T-31.134: URL invalid → warning
test('T-31.134: url invalid', () => {
  const result = validateAttributeValue('url', 'not a url')
  expect(result).toContain('URL')
})

// URL valid → null
test('T-31.134a: url valid', () => {
  expect(validateAttributeValue('url', 'https://example.com')).toBeNull()
})

// URL with file:// scheme (no host) → warning
test('T-31.134a2: url file scheme no host rejected', () => {
  const result = validateAttributeValue('url', 'file:///tmp')
  expect(result).toContain('URL')
  expect(result).toContain('scheme and host')
})

// Date invalid → warning
test('T-31.134b: date invalid format', () => {
  const result = validateAttributeValue('date', 'not-a-date')
  expect(result).toContain('date')
})

// Date valid → null
test('T-31.134c: date valid', () => {
  expect(validateAttributeValue('date', '2026-04-15')).toBeNull()
})

// Date with trailing junk → warning
test('date with trailing junk rejected', () => {
  const result = validateAttributeValue('date', '2026-04-15abc')
  expect(result).not.toBeNull()
  expect(result).toContain('date')
})

// Date with impossible day (Feb 31) → warning
test('T-31.134d: date impossible day rejected', () => {
  const result = validateAttributeValue('date', '2024-02-31')
  expect(result).not.toBeNull()
  expect(result).toContain('date')
})

// Date with impossible month (month 13) → warning
test('T-31.134e: date impossible month rejected', () => {
  const result = validateAttributeValue('date', '2024-13-01')
  expect(result).not.toBeNull()
})

// Leap year Feb 29 valid
test('T-31.134f: date leap year feb 29 valid', () => {
  expect(validateAttributeValue('date', '2024-02-29')).toBeNull()
})

// Non-leap year Feb 29 → warning
test('T-31.134g: date non-leap year feb 29 rejected', () => {
  const result = validateAttributeValue('date', '2025-02-29')
  expect(result).not.toBeNull()
})

// JSON invalid → warning
test('T-31.134h: json invalid', () => {
  const result = validateAttributeValue('json', '{bad}')
  expect(result).toContain('JSON')
})

// JSON valid → null
test('T-31.134i: json valid', () => {
  expect(validateAttributeValue('json', '{"key":"value"}')).toBeNull()
})

// List exceeds max_length → warning
test('T-31.134j: list exceeds max_length', () => {
  const result = validateAttributeValue('list', '["a","b","c"]', { max_length: 2 })
  expect(result).toContain('maximum')
})

// List within max_length → null
test('T-31.134k: list within max_length', () => {
  expect(validateAttributeValue('list', '["a","b"]', { max_length: 5 })).toBeNull()
})

// List invalid json → warning
test('T-31.134l: list invalid json', () => {
  const result = validateAttributeValue('list', 'not json')
  expect(result).toContain('JSON array')
})

// Boolean → no warning (checkbox prevents invalid)
test('T-31.134m: boolean no warning', () => {
  expect(validateAttributeValue('boolean', 'true')).toBeNull()
  expect(validateAttributeValue('boolean', 'false')).toBeNull()
})

// Enum → no warning (select prevents invalid)
test('T-31.134n: enum no warning', () => {
  expect(validateAttributeValue('enum', 'anything')).toBeNull()
})

// No constraints → null
test('T-31.134o: no constraints no warning', () => {
  expect(validateAttributeValue('string', 'anything')).toBeNull()
})

// Non-numeric string for integer/number → warning
test('T-31.134p: integer non-numeric string', () => {
  const result = validateAttributeValue('integer', 'abc')
  expect(result).toContain('number')
})

test('T-31.134q: number non-numeric string', () => {
  const result = validateAttributeValue('number', 'abc')
  expect(result).toContain('number')
})

// List valid JSON but not array → warning
test('T-31.134r: list non-array json', () => {
  const result = validateAttributeValue('list', '{"not":"array"}')
  expect(result).toContain('JSON array')
})

// Integer within range no constraints → null (covers validateMinMax with no constraints)
test('T-31.134s: integer no constraints', () => {
  expect(validateAttributeValue('integer', '42')).toBeNull()
})

// Number no constraints → null (covers validateMinMax with no constraints)
test('T-31.134t: number no constraints', () => {
  expect(validateAttributeValue('number', '3.14')).toBeNull()
})

// List with unknown element_base_type → all elements pass (default branch in isValidElement)
test('list with unknown element type passes all elements', () => {
  expect(validateAttributeValue('list', '[1, "two", true]', { element_base_type: 'custom_type' })).toBeNull()
})

// Copilot review: [0.0] should fail integer list validation (matches backend behavior)
test('list integer rejects float literal 0.0', () => {
  const result = validateAttributeValue('list', '[1, 2, 0.0, 4]', { element_base_type: 'integer' })
  expect(result).toContain('index 2')
  expect(result).toContain('integer')
})

test('list integer rejects scientific notation 1e2', () => {
  const result = validateAttributeValue('list', '[1e2]', { element_base_type: 'integer' })
  expect(result).toContain('index 0')
  expect(result).toContain('integer')
})
