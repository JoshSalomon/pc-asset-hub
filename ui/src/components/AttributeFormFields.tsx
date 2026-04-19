import { FormGroup, TextInput, TextArea, Checkbox, HelperText, HelperTextItem, DatePicker } from '@patternfly/react-core'
import type { SnapshotAttribute } from '../types'
import { validateAttributeValue } from '../utils/validateAttributeValue'

interface Props {
  schemaAttrs: SnapshotAttribute[]
  values: Record<string, string>
  onChange: (name: string, value: string) => void
  enumValues: Record<string, string[]>
  idPrefix: string
  includeSystem?: boolean
  systemName?: string
  setSystemName?: (v: string) => void
  systemDesc?: string
  setSystemDesc?: (v: string) => void
}

export default function AttributeFormFields({
  schemaAttrs, values, onChange, enumValues, idPrefix,
  includeSystem, systemName, setSystemName, systemDesc, setSystemDesc,
}: Props) {
  return (
    <>
      {schemaAttrs.map(attr => {
        if (attr.system && attr.name === 'name') {
          if (!includeSystem) return null
          return (
            <FormGroup key={attr.name} label="Name" isRequired fieldId={`${idPrefix}-name`}>
              <TextInput id={`${idPrefix}-name`} value={systemName || ''} onChange={(_e, v) => setSystemName?.(v)} isRequired />
            </FormGroup>
          )
        }
        if (attr.system && attr.name === 'description') {
          if (!includeSystem) return null
          return (
            <FormGroup key={attr.name} label="Description" fieldId={`${idPrefix}-desc`}>
              <TextInput id={`${idPrefix}-desc`} value={systemDesc || ''} onChange={(_e, v) => setSystemDesc?.(v)} />
            </FormGroup>
          )
        }

        const baseType = attr.base_type || 'string'
        // For enum types, check constraints.values or look up from enumValues cache using type_definition_version_id
        const enumId = attr.type_definition_version_id || ''
        const constraintValues = (attr.constraints?.values as string[]) || []
        const cachedValues = enumId ? enumValues[enumId] : undefined
        const enumOpts = constraintValues.length > 0 ? constraintValues : (cachedValues || [])

        // Inline validation warning (advisory, not blocking)
        const warning = (baseType !== 'enum' && baseType !== 'boolean' && values[attr.name])
          ? validateAttributeValue(baseType, values[attr.name], attr.constraints as Record<string, unknown> | undefined)
          : null
        const validated = warning ? 'warning' as const : 'default' as const

        return (
          <FormGroup key={attr.name} label={`${attr.name}${attr.required ? ' *' : ''}`} fieldId={`${idPrefix}-attr-${attr.name}`}>
            {baseType === 'enum' && enumOpts.length > 0 ? (
              <select
                id={`${idPrefix}-attr-${attr.name}`}
                value={values[attr.name] || ''}
                onChange={(e) => onChange(attr.name, e.target.value)}
                style={{ width: '100%', padding: '6px 12px' }}
              >
                <option value="">Select...</option>
                {enumOpts.map(v => <option key={v} value={v}>{v}</option>)}
              </select>
            ) : baseType === 'boolean' ? (
              <Checkbox
                id={`${idPrefix}-attr-${attr.name}`}
                label="Yes"
                aria-label={attr.name}
                isChecked={values[attr.name] === 'true'}
                onChange={(_e, checked) => onChange(attr.name, checked ? 'true' : 'false')}
              />
            ) : baseType === 'integer' ? (
              <TextInput
                id={`${idPrefix}-attr-${attr.name}`}
                type="number"
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                step={1}
                validated={validated}
              />
            ) : baseType === 'number' ? (
              <input
                id={`${idPrefix}-attr-${attr.name}`}
                type="text"
                inputMode="decimal"
                value={values[attr.name] || ''}
                onChange={(e) => onChange(attr.name, e.target.value)}
                className="pf-v6-c-form-control"
                aria-label={`${attr.name}${attr.required ? ' *' : ''}`}
              />
            ) : baseType === 'date' ? (
              <DatePicker
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                dateFormat={(date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`}
                dateParse={(val: string) => { const [y, m, d] = val.split('-').map(Number); return new Date(y, m - 1, d) }}
                placeholder="YYYY-MM-DD"
                aria-label={attr.name}
              />
            ) : baseType === 'url' ? (
              <TextInput
                id={`${idPrefix}-attr-${attr.name}`}
                type="text"
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                placeholder="https://..."
                validated={validated}
              />
            ) : baseType === 'json' ? (
              <TextArea
                id={`${idPrefix}-attr-${attr.name}`}
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                placeholder='{"key": "value"}'
                rows={3}
                validated={validated}
              />
            ) : baseType === 'list' ? (
              <TextArea
                id={`${idPrefix}-attr-${attr.name}`}
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                placeholder='["value1", "value2"]'
                rows={2}
                validated={validated}
              />
            ) : attr.constraints?.multiline ? (
              <TextArea
                id={`${idPrefix}-attr-${attr.name}`}
                aria-label={attr.name}
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                rows={4}
                validated={validated}
              />
            ) : (
              <TextInput
                id={`${idPrefix}-attr-${attr.name}`}
                type="text"
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
                validated={validated}
              />
            )}
            {warning && (
              <HelperText>
                <HelperTextItem variant="warning">{warning}</HelperTextItem>
              </HelperText>
            )}
          </FormGroup>
        )
      })}
    </>
  )
}
