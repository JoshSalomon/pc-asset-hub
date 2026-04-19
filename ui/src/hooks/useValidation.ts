import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { ValidationError } from '../types'

const STORAGE_PREFIX = 'validation:'

function loadFromSession(catalogName: string | undefined): { errors: ValidationError[]; ran: boolean } | null {
  if (!catalogName) return null
  try {
    const raw = sessionStorage.getItem(`${STORAGE_PREFIX}${catalogName}`)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.ran === 'boolean' && Array.isArray(parsed.errors)
      && parsed.errors.every((e: unknown) => e && typeof e === 'object' && 'entity_type' in e && 'violation' in e)) {
      return { errors: parsed.errors, ran: parsed.ran }
    }
  } catch { /* ignore corrupt storage */ }
  return null
}

function saveToSession(catalogName: string, errors: ValidationError[], ran: boolean) {
  try {
    sessionStorage.setItem(`${STORAGE_PREFIX}${catalogName}`, JSON.stringify({ errors, ran }))
  } catch { /* ignore quota errors */ }
}

export function useValidation(catalogName: string | undefined, onComplete?: () => Promise<void>) {
  const [errors, setErrors] = useState<ValidationError[]>([])
  const [validating, setValidating] = useState(false)
  const [ran, setRan] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Rehydrate from sessionStorage on mount
  useEffect(() => {
    const saved = loadFromSession(catalogName)
    if (saved) {
      setErrors(saved.errors)
      setRan(saved.ran)
    }
  }, [catalogName])

  const validate = async () => {
    if (!catalogName) return
    setValidating(true)
    setErrors([])
    setRan(false)
    setError(null)
    try {
      const result = await api.catalogs.validate(catalogName)
      const newErrors = result.errors || []
      setErrors(newErrors)
      setRan(true)
      saveToSession(catalogName, newErrors, true)
      if (onComplete) await onComplete()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Validation failed')
    } finally {
      setValidating(false)
    }
  }

  return { errors, validating, ran, error, validate }
}
