import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { disconnect, listConnections, startConnect } from "./api"

export const calendarConnectionKeys = {
  list: () => ["calendar", "connections"] as const,
}

export function useCalendarConnections() {
  return useQuery({
    queryKey: calendarConnectionKeys.list(),
    queryFn: listConnections,
  })
}

export function useStartConnect() {
  return useMutation({
    mutationFn: (provider: string) => startConnect(provider),
    onSuccess: (data) => {
      window.location.href = data.auth_url
    },
  })
}

export function useDisconnect() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (provider: string) => disconnect(provider),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: calendarConnectionKeys.list(),
      })
    },
  })
}
