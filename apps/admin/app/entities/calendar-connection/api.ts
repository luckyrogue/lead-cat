import { api } from "~/shared/api/client"

import type { CalendarConnection } from "./types"

export async function listConnections(): Promise<CalendarConnection[]> {
  const { data } = await api.get<CalendarConnection[]>(
    "/api/calendar/connections"
  )
  return data
}

export async function startConnect(
  provider: string
): Promise<{ auth_url: string }> {
  const { data } = await api.post<{ auth_url: string }>(
    `/api/calendar/connect/${provider}/start`,
    {}
  )
  return data
}

export async function disconnect(provider: string): Promise<void> {
  await api.delete(`/api/calendar/connections/${provider}`)
}
