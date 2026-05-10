export function errorMessage(error: unknown, fallback = 'An error occurred'): string {
  return error instanceof Error ? error.message : fallback
}
