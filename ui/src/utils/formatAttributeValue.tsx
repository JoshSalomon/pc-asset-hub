import type { ReactNode } from 'react'

export function formatAttributeValue(type: string, value: string | number | null): ReactNode {
  if (value == null) return '\u2014'

  switch (type) {
    case 'url': {
      const str = String(value)
      if (/^https?:\/\//i.test(str)) {
        return (
          <a href={str} target="_blank" rel="noopener noreferrer">
            {str}
          </a>
        )
      }
      return str
    }

    case 'boolean':
      if (value === 'true') return 'Yes'
      if (value === 'false') return 'No'
      return <>{String(value)} <span aria-hidden="true" title="Unexpected boolean value">⚠</span><span style={{ position: 'absolute', width: '1px', height: '1px', overflow: 'hidden', clipPath: 'inset(50%)', whiteSpace: 'nowrap' }}>Warning: unexpected boolean value</span></>


    case 'date': {
      const d = new Date(String(value))
      return isNaN(d.getTime()) ? String(value) : d.toLocaleDateString()
    }

    case 'json': {
      try {
        const parsed = JSON.parse(String(value))
        return <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{JSON.stringify(parsed, null, 2)}</pre>
      } catch {
        return String(value)
      }
    }

    case 'list': {
      try {
        const arr = JSON.parse(String(value))
        if (Array.isArray(arr)) {
          return arr.join(', ')
        }
        return String(value)
      } catch {
        return String(value)
      }
    }

    default:
      return String(value)
  }
}
