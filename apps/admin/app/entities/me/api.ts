import { api } from "~/shared/api/client"

export type MeSettings = { timezone: string; language: string }

export async function getMeSettings(): Promise<MeSettings> {
  const { data } = await api.get<MeSettings>("/api/auth/web/me/settings")
  return data
}

export async function updateMeSettings(
  prefs: Partial<MeSettings>
): Promise<void> {
  await api.patch("/api/auth/web/me/settings", prefs)
}
