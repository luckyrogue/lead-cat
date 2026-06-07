import type { Scope } from "@/entities/meeting/api"

export type MeetingsFilter = "up" | "past" | "all"

export function parseMeetingsFilter(
  searchParams: URLSearchParams
): MeetingsFilter {
  return scopeToFilter(searchParams.get("scope"))
}

export function scopeToFilter(scope: string | null): MeetingsFilter {
  if (scope === "past") return "past"
  if (scope === "all") return "all"
  return "up"
}

export function filterToScope(filter: MeetingsFilter): string {
  if (filter === "past") return "past"
  if (filter === "all") return "all"
  return "upcoming"
}

export function serializeMeetingsFilter(filter: MeetingsFilter): string {
  return filterToScope(filter)
}

export function meetingsScopeFromFilter(filter: MeetingsFilter): Scope {
  if (filter === "past") return "past"
  if (filter === "all") return "all"
  return "upcoming"
}
