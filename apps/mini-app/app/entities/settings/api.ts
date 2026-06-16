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

export const TIMEZONE_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "Default (Almaty)" },
  { value: "Asia/Almaty", label: "Almaty (UTC+5)" },
  { value: "Asia/Tashkent", label: "Tashkent (UTC+5)" },
  { value: "Asia/Bishkek", label: "Bishkek (UTC+6)" },
  { value: "Europe/Moscow", label: "Moscow (UTC+3)" },
  { value: "Europe/Kyiv", label: "Kyiv (UTC+2/3)" },
  { value: "Europe/London", label: "London (UTC+0/1)" },
  { value: "Asia/Dubai", label: "Dubai (UTC+4)" },
  { value: "Asia/Istanbul", label: "Istanbul (UTC+3)" },
  { value: "America/New_York", label: "New York (UTC-5/4)" },
  { value: "UTC", label: "UTC" },
]

export const LANGUAGE_OPTIONS: { value: string; label: string }[] = [
  { value: "", label: "Default" },
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
  { value: "kk", label: "Қазақша" },
]

export async function fetchSettings(): Promise<UserSettings> {
  const res = await apiFetch<UserSettings>("/api/miniapp/settings")
  return {
    reminder_minutes: res.reminder_minutes ?? [],
    timezone: res.timezone ?? "",
    language: res.language ?? "",
  }
}

export async function updateReminderMinutes(minutes: number[]): Promise<void> {
  await apiFetch<void>("/api/miniapp/settings", {
    method: "PATCH",
    body: { reminder_minutes: minutes },
  })
}

export async function updatePrefs(prefs: {
  timezone?: string
  language?: string
}): Promise<void> {
  await apiFetch<void>("/api/miniapp/settings", {
    method: "PATCH",
    body: prefs,
  })
}
