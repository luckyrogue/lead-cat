import type { MiniAppUserSettings } from "@leadcat/api-client"

import { apiFetch } from "~/shared/api/client"

export type UserSettings = MiniAppUserSettings

export const REMINDER_OPTIONS: { minutes: number; label: string }[] = [
  { minutes: 10, label: "10m" },
  { minutes: 15, label: "15m" },
  { minutes: 30, label: "30m" },
  { minutes: 60, label: "1h" },
  { minutes: 120, label: "2h" },
  { minutes: 1440, label: "1d" },
]

export async function fetchSettings(): Promise<UserSettings> {
  const res = await apiFetch<UserSettings>("/api/miniapp/settings")
  return { reminder_minutes: res.reminder_minutes ?? [] }
}

export async function updateReminderMinutes(minutes: number[]): Promise<void> {
  await apiFetch<void>("/api/miniapp/settings", {
    method: "PATCH",
    body: { reminder_minutes: minutes },
  })
}
