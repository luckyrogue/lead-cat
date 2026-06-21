import { ApiError } from "./errors"

export function mapApiErrorMessage(
  t: (key: string) => string,
  error: unknown,
  fallbackKey: string
): string {
  const fallback = t(fallbackKey)

  if (error instanceof ApiError) {
    const code = error.code
    if (code) {
      const codeKey = `errors.codes.${code}`
      const mapped = t(codeKey)
      if (mapped !== codeKey) {
        return mapped
      }
    }
    // Only surface the raw backend message for client (4xx) errors, which are
    // user-facing validation messages — and only when it's human text rather
    // than a bare machine code. Never leak internal 5xx error strings.
    if (error.status < 500 && error.message && error.message !== code) {
      return error.message
    }
    return fallback
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}
