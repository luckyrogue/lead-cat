import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query"

import {
  fetchSettings,
  updatePrefs,
  updateReminderMinutes,
  type UserSettings,
} from "~/entities/settings/api"
import { settingsKeys } from "~/shared/api/query-keys"

export function settingsQuery() {
  return queryOptions({
    queryKey: settingsKeys.all,
    queryFn: fetchSettings,
    staleTime: 60_000,
  })
}

export function useUpdateReminderMinutes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (minutes: number[]) => updateReminderMinutes(minutes),
    onSuccess: (_data, minutes) => {
      qc.setQueryData<UserSettings>(settingsKeys.all, (current) =>
        current ? { ...current, reminder_minutes: minutes } : current
      )
    },
  })
}

export function useUpdatePrefs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (prefs: { timezone?: string; language?: string }) =>
      updatePrefs(prefs),
    onSuccess: (_data, prefs) => {
      qc.setQueryData<UserSettings>(settingsKeys.all, (current) =>
        current
          ? {
              ...current,
              ...(prefs.timezone !== undefined
                ? { timezone: prefs.timezone }
                : {}),
              ...(prefs.language !== undefined
                ? { language: prefs.language }
                : {}),
            }
          : current
      )
    },
  })
}
