import type {
  BookingEventType,
  BookingEventTypesResponse,
} from "@leadcat/api-client"

import { api } from "~/shared/api/client"
import type { EventTypeInput } from "~/entities/booking-event-type/types"

export async function listEventTypes(): Promise<BookingEventType[]> {
  const { data } = await api.get<BookingEventTypesResponse>(
    "/api/booking/event-types"
  )
  return data.event_types ?? []
}

export async function createEventType(
  input: EventTypeInput
): Promise<BookingEventType> {
  const { data } = await api.post<BookingEventType>(
    "/api/booking/event-types",
    input
  )
  return data
}

export async function updateEventType(
  id: string,
  input: EventTypeInput
): Promise<void> {
  await api.patch(`/api/booking/event-types/${id}`, input)
}

export async function deleteEventType(id: string): Promise<void> {
  await api.delete(`/api/booking/event-types/${id}`)
}
