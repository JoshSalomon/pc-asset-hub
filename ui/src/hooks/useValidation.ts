import { useState } from 'react'
import { api } from '../api/client'
import type { ValidationError } from '../types'

export function useValidation(catalogName: string | undefined, onComplete?: () => Promise<void>) {
  const [errors, setErrors] = useState<ValidationError[]>([])
  const [validating, setValidating] = useState(false)
  const [ran, setRan] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const validate = async () => {
    if (!catalogName) return
    setValidating(true)
    setErrors([])
    setRan(false)
    setError(null)
    try {
      const result = await api.catalogs.validate(catalogName)
      setErrors(result.errors || [])
      setRan(true)
      if (onComplete) await onComplete()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Validation failed')
    } finally {
      setValidating(false)
    }
  }

  return { errors, validating, ran, error, validate }
}
