import { Alert } from '@patternfly/react-core'
import type { ValidationError } from '../types'

export default function ValidationResults({ errors, ran, error }: { errors: ValidationError[]; ran: boolean; error?: string | null }) {
  if (error) {
    return <Alert variant="warning" title={`Validation request failed: ${error}`} isInline style={{ marginBottom: '1rem' }} />
  }

  if (!ran) return null

  if (errors.length === 0) {
    return <Alert variant="success" title="Validation passed — catalog is valid" isInline style={{ marginBottom: '1rem' }} />
  }

  return (
    <Alert variant="danger" title={`Validation failed — ${errors.length} error(s) found`} isInline style={{ marginBottom: '1rem' }}>
      <div data-testid="validation-error-list" style={{ maxHeight: '300px', overflowY: 'auto' }}>
        {Object.entries(
          errors.reduce<Record<string, ValidationError[]>>((acc, e) => {
            (acc[e.entity_type] = acc[e.entity_type] || []).push(e)
            return acc
          }, {})
        ).map(([entityType, errs]) => (
          <div key={entityType} style={{ marginTop: '0.5rem' }}>
            <strong>{entityType}</strong>
            <ul style={{ margin: '0.25rem 0', paddingLeft: '1.5rem' }}>
              {errs.map((e) => (
                <li key={`${e.entity_type}:${e.instance_name}:${e.field}:${e.violation}`}>{e.instance_name}: {e.field} — {e.violation}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
    </Alert>
  )
}
