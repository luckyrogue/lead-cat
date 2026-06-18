import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { disconnectCalendar, listConnections, startConnect } from "~/entities/calendar-connection/api"
import type { CalendarProvider } from "~/entities/calendar-connection/types"
import { calendarConnectionKeys } from "~/shared/api/query-keys"

export function calendarConnectionsQuery() {
  return queryOptions({
    queryKey: calendarConnectionKeys.all,
    queryFn: listConnections,
  })
}

export function useCalendarConnections() {
  return useQuery(calendarConnectionsQuery())
}

export function useStartConnect() {
  return useMutation({
    mutationFn: (provider: CalendarProvider) => startConnect(provider),
  })
}

export function useDisconnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (provider: CalendarProvider) => disconnectCalendar(provider),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: calendarConnectionKeys.all })
    },
  })
}
