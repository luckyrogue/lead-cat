import { buildListSearchParams } from "@/shared/lib/list-url-params"
import {
  filterToScope,
  scopeToFilter,
  type MeetingsFilter,
} from "@/features/meetings/queries"

export function parseMeetingsScopeFilter(
  searchParams: URLSearchParams
): MeetingsFilter {
  return scopeToFilter(searchParams.get("scope"))
}

export function buildMeetingsSearchParams(input: {
  q: string
  page: number
  filter: MeetingsFilter
  success?: string
}) {
  const params = buildListSearchParams({
    q: input.q,
    page: input.page,
    filterKey: "scope",
    filterValue: filterToScope(input.filter),
  })
  if (input.success) params.set("success", input.success)
  return params
}
