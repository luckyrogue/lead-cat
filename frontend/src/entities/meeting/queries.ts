import { queryOptions, useQuery } from "@tanstack/react-query"
import {
  fetchColleagueSchedule,
  fetchMyMeetings,
  type Scope,
} from "@/entities/meeting/api"
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
