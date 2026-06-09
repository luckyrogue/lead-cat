import { apiFetch } from "@/shared/api/client"
import type { UserSettings } from "./types"

type DTO = { reminder_minutes: number[] }

export async function getUserSettings(): Promise<UserSettings> {
  const d = await apiFetch<DTO>("/api/miniapp/settings")
  return { reminderMinutes: d.reminder_minutes ?? [] }
}
