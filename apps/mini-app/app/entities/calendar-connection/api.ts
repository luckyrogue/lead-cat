import { apiFetch } from "~/shared/api/client"
import type {
  CalendarConnection,
  CalendarProvider,
  StartConnectResponse,
} from "~/entities/calendar-connection/types"

export async function listConnections(): Promise<CalendarConnection[]> {
  const res = await apiFetch<CalendarConnection[]>(
    "/api/miniapp/calendar/connections"
  )
  return res ?? []
}

export async function startConnect(
  provider: CalendarProvider
): Promise<StartConnectResponse> {
  return apiFetch<StartConnectResponse>(
    `/api/miniapp/calendar/connect/${provider}/start`,
    {
      method: "POST",
    }
  )
}

export async function disconnectCalendar(
  provider: CalendarProvider
): Promise<void> {
  await apiFetch<void>(`/api/miniapp/calendar/connections/${provider}`, {
    method: "DELETE",
  })
}
