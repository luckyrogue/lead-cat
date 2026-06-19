import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createEventType,
  deleteEventType,
  listEventTypes,
  updateEventType,
} from "~/entities/booking-event-type/api"
import type { EventTypeInput } from "~/entities/booking-event-type/types"

export const eventTypeKeys = {
  list: () => ["booking", "event-types"] as const,
}

export function useMyEventTypes() {
  return useQuery({
    queryKey: eventTypeKeys.list(),
    queryFn: listEventTypes,
  })
}

export function useCreateEventType() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: EventTypeInput) => createEventType(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list() })
    },
  })
}

export function useUpdateEventType() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; input: EventTypeInput }) =>
      updateEventType(args.id, args.input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list() })
    },
  })
}

export function useDeleteEventType() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteEventType(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventTypeKeys.list() })
    },
  })
}
