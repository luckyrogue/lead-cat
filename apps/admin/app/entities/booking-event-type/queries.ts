import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createEventType,
  deleteEventType,
  listEventTypes,
  updateEventType,
} from "~/entities/booking-event-type/api"
import type { EventTypeInput } from "~/entities/booking-event-type/types"

export const eventTypeKeys = {
  list: (orgId: string) => ["orgs", orgId, "booking", "event-types"] as const,
}

export function useMyEventTypes(orgId: string | null) {
  return useQuery({
    queryKey: eventTypeKeys.list(orgId ?? ""),
    queryFn: listEventTypes,
    enabled: Boolean(orgId),
  })
}

export function useCreateEventType(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: EventTypeInput) => createEventType(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list(orgId) })
    },
  })
}

export function useUpdateEventType(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; input: EventTypeInput }) =>
      updateEventType(args.id, args.input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list(orgId) })
    },
  })
}

export function useDeleteEventType(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteEventType(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list(orgId) })
    },
  })
}
