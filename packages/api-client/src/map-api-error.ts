import { ApiError } from "./errors"

export function mapApiErrorMessage(
  t: (key: string) => string,
  error: unknown,
  fallbackKey: string
): string {
  const fallback = t(fallbackKey)

  if (error instanceof ApiError) {
    const code = error.code ?? error.message
    if (code) {
      const codeKey = `errors.codes.${code}`
      const mapped = t(codeKey)
      if (mapped !== codeKey) {
        return mapped
      }
    }
    if (error.message && error.message !== code) {
      return error.message
    }
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}
