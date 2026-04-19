import { useState } from 'react'
import {
  Select,
  SelectOption,
  SelectGroup,
  MenuToggle,
  TextInputGroup,
  TextInputGroupMain,
  type MenuToggleElement,
} from '@patternfly/react-core'
import type { TypeDefinition } from '../types'
import { typeLabel } from '../utils/typeLabel'

interface Props {
  typeDefinitions: TypeDefinition[]
  selectedTdId: string
  onSelect: (tdId: string) => void
}

export default function TypeDefinitionSelector({ typeDefinitions, selectedTdId, onSelect }: Props) {
  const [tdOpen, setTdOpen] = useState(false)
  const [filterValue, setFilterValue] = useState('')

  const lowerFilter = filterValue.toLowerCase()
  const systemTypes = typeDefinitions.filter(td => td.system && (!lowerFilter || td.name.toLowerCase().includes(lowerFilter)))
  const customTypes = typeDefinitions.filter(td => !td.system && (!lowerFilter || td.name.toLowerCase().includes(lowerFilter)))

  const selectedTd = typeDefinitions.find(t => t.id === selectedTdId)
  const toggleText = selectedTd ? typeLabel(selectedTd) : 'Select type...'

  return (
    <Select
      isOpen={tdOpen}
      selected={selectedTdId}
      onSelect={(_e, value) => { onSelect(value as string); setTdOpen(false); setFilterValue('') }}
      onOpenChange={(open) => { setTdOpen(open); if (!open) setFilterValue('') }}
      toggle={(ref: React.Ref<MenuToggleElement>) => (
        <MenuToggle ref={ref} onClick={() => setTdOpen(!tdOpen)} isExpanded={tdOpen}>
          {toggleText}
        </MenuToggle>
      )}
    >
      <div style={{ padding: '0.5rem' }}>
        <TextInputGroup>
          <TextInputGroupMain
            value={filterValue}
            onChange={(_e, v) => setFilterValue(v)}
            placeholder="Filter types..."
            role="searchbox"
            autoFocus
          />
        </TextInputGroup>
      </div>
      <div style={{ maxHeight: '200px', overflow: 'auto' }} data-testid="type-select-scroll">
        {systemTypes.length > 0 && (
          <SelectGroup label="System Types">
            {systemTypes.map(td => (
              <SelectOption key={td.id} value={td.id}>
                {typeLabel(td)}
              </SelectOption>
            ))}
          </SelectGroup>
        )}
        {customTypes.length > 0 && (
          <SelectGroup label="Custom Types">
            {customTypes.map(td => (
              <SelectOption key={td.id} value={td.id}>
                {typeLabel(td)}
              </SelectOption>
            ))}
          </SelectGroup>
        )}
      </div>
    </Select>
  )
}
