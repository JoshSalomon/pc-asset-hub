const regexCache = new Map<string, RegExp>()

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
      const key = `^(?:${pattern})$`
      let re = regexCache.get(key)
      if (!re) {
        re = new RegExp(key)
        regexCache.set(key, re)
      }
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

function isValidElement(elemType: string, item: unknown): boolean {
  switch (elemType) {
    case 'string':
      return typeof item === 'string'
    case 'number':
      return typeof item === 'number'
    case 'integer':
      return typeof item === 'number' && Number.isInteger(item)
    case 'boolean':
      return typeof item === 'boolean'
    default:
      return true
  }
}

function validateList(value: string, constraints?: Record<string, unknown>): string | null {
  let arr: unknown
  try {
    arr = JSON.parse(value)
  } catch {
    return 'Enter values as a JSON array, e.g. [1, 2, 3]'
  }
  if (!Array.isArray(arr)) return 'Enter values as a JSON array, e.g. [1, 2, 3]'
  if (constraints) {
    const maxLength = constraints.max_length
    if (typeof maxLength === 'number' && arr.length > maxLength) {
      return `List exceeds maximum of ${maxLength} items`
    }
    const elemType = constraints.element_base_type
    if (typeof elemType === 'string') {
      const rawTokens = elemType === 'integer' ? extractArrayTokens(value) : null
      for (let i = 0; i < arr.length; i++) {
        if (elemType === 'integer' && rawTokens && rawTokens[i] && /[.eE]/.test(rawTokens[i])) {
          return `Element at index ${i} must be a ${elemType}`
        }
        if (!isValidElement(elemType, arr[i])) {
          return `Element at index ${i} must be a ${elemType}`
        }
      }
    }
  }
  return null
}

function extractArrayTokens(json: string): string[] {
  const inner = json.replace(/^\s*\[/, '').replace(/]\s*$/, '')
  return inner.split(',').map(t => t.trim())
}
