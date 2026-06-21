import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { getMeSettings, updateMeSettings } from "~/entities/me/api"
import type { MeSettings } from "~/entities/me/api"
import type { Me } from "~/shared/auth/types"
import { meQueryKey } from "~/shared/auth/use-me"

export const meSettingsQueryKey = ["auth", "me-settings"] as const

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
    onSuccess: (_data, prefs) => {
      queryClient.setQueryData<MeSettings>(meSettingsQueryKey, (current) => ({
        timezone: prefs.timezone ?? current?.timezone ?? "",
        language: prefs.language ?? current?.language ?? "",
      }))
      queryClient.setQueryData<Me | null>(meQueryKey, (current) => {
        if (!current) {
          return current
        }
        return {
          ...current,
          user: {
            ...current.user,
            ...(prefs.timezone !== undefined
              ? { timezone: prefs.timezone }
              : {}),
            ...(prefs.language !== undefined
              ? { language: prefs.language }
              : {}),
          },
        }
      })
    },
  })
}
