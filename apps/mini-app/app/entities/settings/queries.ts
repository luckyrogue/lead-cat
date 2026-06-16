import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"

import { fetchSettings, updatePrefs, updateReminderMinutes } from "~/entities/settings/api"
import { settingsKeys } from "~/shared/api/query-keys"

export function settingsQuery() {
  return queryOptions({
    queryKey: settingsKeys.all,
    queryFn: fetchSettings,
  })
}

export function useUpdateReminderMinutes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (minutes: number[]) => updateReminderMinutes(minutes),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: settingsKeys.all })
    },
  })
}

export function useUpdatePrefs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (prefs: { timezone?: string; language?: string }) =>
      updatePrefs(prefs),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: settingsKeys.all })
    },
  })
}
