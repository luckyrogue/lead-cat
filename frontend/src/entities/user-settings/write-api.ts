import { apiFetch } from "@/shared/api/client"

export async function patchUserSettings(reminderMinutes: number[]): Promise<void> {
  await apiFetch("/api/miniapp/settings", {
    method: "PATCH",
    body: JSON.stringify({ reminder_minutes: reminderMinutes }),
  })
}
