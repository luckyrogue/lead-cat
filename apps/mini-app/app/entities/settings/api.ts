import type { MiniAppUserSettings } from "@leadcat/api-client"

import { apiFetch } from "~/shared/api/client"

export type UserSettings = MiniAppUserSettings

const REMINDER_MINUTES = [10, 15, 30, 60, 120, 1440] as const

const TIMEZONE_KEYS = [
  { value: "Asia/Almaty", key: "almaty" },
  { value: "Asia/Tashkent", key: "tashkent" },
  { value: "Asia/Bishkek", key: "bishkek" },
  { value: "Europe/Moscow", key: "moscow" },
  { value: "Europe/Kyiv", key: "kyiv" },
  { value: "Europe/London", key: "london" },
  { value: "Asia/Dubai", key: "dubai" },
  { value: "Asia/Istanbul", key: "istanbul" },
  { value: "America/New_York", key: "newYork" },
  { value: "UTC", key: "utc" },
] as const

const LANGUAGE_VALUES = ["", "ru", "en", "kk"] as const

type TFn = (key: string) => string

export function getReminderOptions(t: TFn) {
  return REMINDER_MINUTES.map((minutes) => ({
    minutes,
    label: t(`profile.reminder.options.${minutes}`),
  }))
}

export function getTimezoneOptions(t: TFn) {
  return [
    { value: "", label: t("profile.preferences.tzDefault") },
    ...TIMEZONE_KEYS.map(({ value, key }) => ({
      value,
      label: t(`timezones.${key}`),
    })),
  ]
}

export function getLanguageOptions(t: TFn) {
  return LANGUAGE_VALUES.map((value) => ({
    value,
    label:
      value === ""
        ? t("profile.preferences.langDefault")
        : t(`profile.preferences.languages.${value}`),
  }))
}

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
