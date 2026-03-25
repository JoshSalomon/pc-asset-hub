import { useState } from 'react'
import {
  Select,
  SelectOption,
  MenuToggle,
  type MenuToggleElement,
} from '@patternfly/react-core'

interface Props {
  id: string
  value: string
  options: string[]
  onChange: (v: string) => void
}

export default function EnumSelect({ id, value, options, onChange }: Props) {
  const [isOpen, setIsOpen] = useState(false)
  return (
    <Select
      id={id}
      isOpen={isOpen}
      selected={value}
      onSelect={(_e, val) => { onChange(val as string); setIsOpen(false) }}
      onOpenChange={setIsOpen}
      toggle={(ref: React.Ref<MenuToggleElement>) => (
        <MenuToggle ref={ref} onClick={() => setIsOpen(!isOpen)} isExpanded={isOpen} style={{ width: '100%' }}>
          {value || 'Select...'}
        </MenuToggle>
      )}
    >
      {options.map(opt => (
        <SelectOption key={opt} value={opt}>{opt}</SelectOption>
      ))}
    </Select>
  )
}
