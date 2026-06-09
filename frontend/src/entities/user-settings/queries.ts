import { useQuery } from "@tanstack/react-query"
import { getUserSettings } from "./api"

export const userSettingsKeys = {
  all: ["user-settings"] as const,
  current: () => ["user-settings", "current"] as const,
}

export function useUserSettings() {
  return useQuery({
    queryKey: userSettingsKeys.current(),
    queryFn: getUserSettings,
  })
}
