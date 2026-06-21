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

// Merge the just-saved prefs into the cached settings read model.
export function patchMeSettings(
  current: MeSettings | undefined,
  prefs: Partial<MeSettings>
): MeSettings {
  return {
    timezone: prefs.timezone ?? current?.timezone ?? "",
    language: prefs.language ?? current?.language ?? "",
  }
}

// Mirror the just-saved prefs onto the cached `me` user without a refetch, so
// the locale/timezone the app reads from `me` flips instantly. Only the fields
// actually present in `prefs` are touched; all other user fields are preserved.
export function patchMeUser(
  current: Me | null | undefined,
  prefs: Partial<MeSettings>
): Me | null | undefined {
  if (!current) {
    return current
  }
  return {
    ...current,
    user: {
      ...current.user,
      ...(prefs.timezone !== undefined ? { timezone: prefs.timezone } : {}),
      ...(prefs.language !== undefined ? { language: prefs.language } : {}),
    },
  }
}

export function useUpdateMeSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (prefs: Partial<MeSettings>) => updateMeSettings(prefs),
    onSuccess: (_data, prefs) => {
      queryClient.setQueryData<MeSettings>(meSettingsQueryKey, (current) =>
        patchMeSettings(current, prefs)
      )
      queryClient.setQueryData<Me | null>(meQueryKey, (current) =>
        patchMeUser(current, prefs)
      )
    },
  })
}
