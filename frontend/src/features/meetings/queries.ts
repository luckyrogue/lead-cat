import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import {
  createMeeting,
  deleteMeeting,
  fetchColleagueSchedule,
  fetchConflicts,
  fetchMyMeetings,
  type ConflictsParams,
  type MeetingInput,
  type MeetingPatch,
  type Scope,
  updateMeeting,
} from "@/features/meetings/api"
import { tmaKeys } from "@/shared/api/query-keys"

export const myMeetingsQuery = (scope: Scope) =>
  queryOptions({
    queryKey: tmaKeys.meetings(scope),
    queryFn: () => fetchMyMeetings(scope),
  })

export const colleagueScheduleQuery = (email: string, scope: Scope) =>
  queryOptions({
    queryKey: tmaKeys.schedule(email, scope),
    queryFn: () => fetchColleagueSchedule(email, scope),
    enabled: email.trim().length > 0,
  })

export function useMyMeetings(scope: Scope) {
  return useQuery(myMeetingsQuery(scope))
}

export function useColleagueSchedule(email: string, scope: Scope) {
  return useQuery(colleagueScheduleQuery(email, scope))
}

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

export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MeetingInput) => createMeeting(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: MeetingPatch }) =>
      updateMeeting(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMeeting(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useConflicts() {
  return useMutation({
    mutationFn: (params: ConflictsParams) => fetchConflicts(params),
  })
}
