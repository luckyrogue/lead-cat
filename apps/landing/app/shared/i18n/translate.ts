import type { Dict } from "~/shared/i18n/types"

function lookup(dict: Dict, key: string): string | undefined {
  const value = key.split(".").reduce<unknown>((acc, part) => {
    if (acc && typeof acc === "object" && part in (acc as Dict)) {
      return (acc as Dict)[part]
    }
    return undefined
  }, dict)
  return typeof value === "string" ? value : undefined
}

function interpolate(
  text: string,
  params?: Record<string, string | number>
): string {
  if (!params) {
    return text
  }
  return text.replace(/\{(\w+)\}/g, (_, k) =>
    params[k] !== undefined ? String(params[k]) : `{${k}}`
  )
}

export function translate(
  active: Dict,
  fallback: Dict,
  key: string,
  params?: Record<string, string | number>
): string {
  const value = lookup(active, key) ?? lookup(fallback, key) ?? key
  return interpolate(value, params)
}
