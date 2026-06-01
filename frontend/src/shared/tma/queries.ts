import { useMutation, useQuery } from "@tanstack/react-query"
import { useTmaApp } from "./context"
import {
  fetchColleagueSchedule,
  fetchFreeSlots,
  fetchMyMeetings,
  searchEmployees,
  type FreeSlotsParams,
  type Scope,
} from "./api"

export function useMyMeetings(scope: Scope) {
  return useQuery({
    queryKey: ["tma", "meetings", scope],
    queryFn: () => fetchMyMeetings(scope),
  })
}

export function useEmployeeSearch(q: string) {
  return useQuery({
    queryKey: ["tma", "employees", q],
    queryFn: () => searchEmployees(q),
    enabled: q.trim().length > 0,
  })
}

export function useColleagueSchedule(email: string, scope: Scope) {
  return useQuery({
    queryKey: ["tma", "schedule", email, scope],
    queryFn: () => fetchColleagueSchedule(email, scope),
    enabled: email.trim().length > 0,
  })
}

export function useFreeSlots() {
  const { lang } = useTmaApp()
  return useMutation({
    mutationFn: (params: FreeSlotsParams) => fetchFreeSlots(params, lang),
  })
}
