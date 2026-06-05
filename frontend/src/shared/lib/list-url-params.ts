import { DEFAULT_LIST_PAGE } from "@/shared/api/list-params"

export function parseListPage(searchParams: URLSearchParams): number {
  const raw = Number(searchParams.get("page"))
  return Number.isFinite(raw) && raw >= 1 ? Math.floor(raw) : DEFAULT_LIST_PAGE
}

export function parseListQuery(searchParams: URLSearchParams): string {
  return searchParams.get("q") ?? ""
}

export function parseCommaList(value: string | null): string[] {
  if (!value?.trim()) {
    return []
  }
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
}

export function serializeCommaList(values: readonly string[]): string | null {
  if (values.length === 0) {
    return null
  }
  return values.join(",")
}

export function areSearchParamsEqual(
  a: URLSearchParams,
  b: URLSearchParams
): boolean {
  const keys = new Set([...a.keys(), ...b.keys()])
  for (const key of keys) {
    if (a.get(key) !== b.get(key)) return false
  }
  return true
}

export function buildListSearchParams(input: {
  q?: string
  page?: number
  filterKey?: string
  filterValue?: string | null
  status?: readonly string[]
}): URLSearchParams {
  const params = new URLSearchParams()
  const trimmedQ = input.q?.trim()
  if (trimmedQ) params.set("q", trimmedQ)
  if (input.filterKey && input.filterValue) {
    params.set(input.filterKey, input.filterValue)
  }
  if (input.status?.length) {
    params.set("status", serializeCommaList(input.status) ?? "")
  }
  if (input.page && input.page > 1) {
    params.set("page", String(input.page))
  }
  return params
}
