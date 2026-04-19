import type { TypeDefinition } from '../types'

export function typeLabel(td: TypeDefinition): string {
  return td.system ? td.name : `${td.name} (${td.base_type})`
}
