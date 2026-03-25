import { FormGroup, TextInput } from '@patternfly/react-core'
import type { SnapshotAttribute } from '../types'
import EnumSelect from './EnumSelect'

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
        return (
          <FormGroup key={attr.name} label={`${attr.name}${attr.required ? ' *' : ''}`} fieldId={`${idPrefix}-attr-${attr.name}`}>
            {attr.type === 'enum' && attr.enum_id && enumValues[attr.enum_id] ? (
              <EnumSelect
                id={`${idPrefix}-attr-${attr.name}`}
                value={values[attr.name] || ''}
                options={enumValues[attr.enum_id]}
                onChange={(v) => onChange(attr.name, v)}
              />
            ) : (
              <TextInput
                id={`${idPrefix}-attr-${attr.name}`}
                type={attr.type === 'number' ? 'number' : 'text'}
                value={values[attr.name] || ''}
                onChange={(_e, v) => onChange(attr.name, v)}
              />
            )}
          </FormGroup>
        )
      })}
    </>
  )
}
