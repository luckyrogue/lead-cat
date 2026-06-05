export type BuildListParamsOptions = {
  qMinLength?: number
}

export function buildListParams<T extends Record<string, unknown>>(
  input: T,
  options: BuildListParamsOptions = {}
): Partial<T> {
  const qMinLength = options.qMinLength ?? 2
  const result: Record<string, unknown> = {}

  for (const [key, value] of Object.entries(input)) {
    if (value === undefined || value === null) {
      continue
    }
    if (key === "q") {
      const trimmed = String(value).trim()
      if (trimmed.length < qMinLength) {
        continue
      }
      result[key] = trimmed
      continue
    }
    if (Array.isArray(value)) {
      if (value.length === 0) {
        continue
      }
      result[key] = value
      continue
    }
    if (typeof value === "string" && value.trim() === "") {
      continue
    }
    result[key] = value
  }

  return result as Partial<T>
}

export const DEFAULT_LIST_PAGE = 1
export const DEFAULT_LIST_PER_PAGE = 20
