import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { getMeSettings, updateMeSettings } from "~/entities/me/api"
import type { MeSettings } from "~/entities/me/api"
import { meQueryKey } from "~/shared/auth/use-me"

export const meSettingsQueryKey = ["auth", "me", "settings"] as const

export function useMeSettings() {
  return useQuery({
    queryKey: meSettingsQueryKey,
    queryFn: getMeSettings,
    staleTime: 60_000,
  })
}

export function useUpdateMeSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (prefs: Partial<MeSettings>) => updateMeSettings(prefs),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meSettingsQueryKey })
      queryClient.invalidateQueries({ queryKey: meQueryKey })
    },
  })
}
