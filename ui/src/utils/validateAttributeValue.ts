export function validateAttributeValue(
  baseType: string,
  value: string,
  constraints?: Record<string, unknown>,
): string | null {
  if (!value) return null // empty = draft mode, no warning

  switch (baseType) {
    case 'string':
      return validateString(value, constraints)
    case 'integer':
      return validateInteger(value, constraints)
    case 'number':
      return validateNumber(value, constraints)
    case 'url':
      return validateUrl(value)
    case 'date':
      return validateDate(value)
    case 'json':
      return validateJson(value)
    case 'list':
      return validateList(value, constraints)
    default:
      return null // boolean, enum — no warnings (controlled inputs)
  }
}

function validateString(value: string, constraints?: Record<string, unknown>): string | null {
  if (!constraints) return null
  const maxLength = constraints.max_length
  if (typeof maxLength === 'number' && value.length > maxLength) {
    return `Exceeds maximum length of ${maxLength}`
  }
  const pattern = constraints.pattern
  if (typeof pattern === 'string') {
    try {
      // Auto-anchor: wrap in ^(?:...)$ so the pattern must match the entire string.
      // Users write patterns like [0-9A-F]+ expecting full-string match;
      // without anchoring, RegExp.test does partial matching.
      const re = new RegExp(`^(?:${pattern})$`)
      if (!re.test(value)) {
        return `Does not match pattern ${pattern}`
      }
    } catch {
      return `Invalid pattern: ${pattern}`
    }
  }
  return null
}

function validateInteger(value: string, constraints?: Record<string, unknown>): string | null {
  const num = Number(value)
  if (isNaN(num)) return 'Must be a valid number'
  if (!Number.isInteger(num)) return 'Must be a whole number'
  return validateMinMax(num, constraints)
}

function validateNumber(value: string, constraints?: Record<string, unknown>): string | null {
  const num = Number(value)
  if (isNaN(num)) return 'Must be a valid number'
  return validateMinMax(num, constraints)
}

function validateMinMax(num: number, constraints?: Record<string, unknown>): string | null {
  if (!constraints) return null
  const min = constraints.min
  if (typeof min === 'number' && num < min) {
    return `Below minimum of ${min}`
  }
  const max = constraints.max
  if (typeof max === 'number' && num > max) {
    return `Above maximum of ${max}`
  }
  return null
}

function validateUrl(value: string): string | null {
  try {
    const u = new URL(value)
    // Defensive: new URL() always sets protocol, but check host to reject e.g. "http://"
    if (!u.protocol || !u.host) return 'Invalid URL: must include scheme and host'
    return null
  } catch {
    return 'Invalid URL'
  }
}

function validateDate(value: string): string | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if (!match) return 'Invalid date format (expected YYYY-MM-DD)'
  const year = Number(match[1])
  const month = Number(match[2]) - 1
  const day = Number(match[3])
  const d = new Date(year, month, day)
  if (isNaN(d.getTime())) return 'Invalid date format (expected YYYY-MM-DD)'
  // Round-trip check: if the date rolled (e.g., Feb 31 → Mar 3), it was invalid
  if (d.getFullYear() !== year || d.getMonth() !== month || d.getDate() !== day) {
    return 'Invalid date (day does not exist in that month)'
  }
  return null
}

function validateJson(value: string): string | null {
  try {
    JSON.parse(value)
    return null
  } catch {
    return 'Invalid JSON syntax'
  }
}

// Note: element_base_type validation is intentionally omitted here. The backend
// (constraint_validator.go) is authoritative and validates each element's type.
// This frontend validator is advisory only — it checks structure and max_length
// but not per-element types. See TD-107 for adding element type checks.
function validateList(value: string, constraints?: Record<string, unknown>): string | null {
  try {
    const arr = JSON.parse(value)
    if (!Array.isArray(arr)) return 'Invalid list: must be a JSON array'
    if (constraints) {
      const maxLength = constraints.max_length
      if (typeof maxLength === 'number' && arr.length > maxLength) {
        return `List exceeds maximum of ${maxLength} items`
      }
    }
    return null
  } catch {
    return 'Invalid list: must be a JSON array'
  }
}
