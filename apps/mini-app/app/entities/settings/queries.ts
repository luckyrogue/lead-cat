import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query"

import { fetchSettings, updateReminderMinutes } from "~/entities/settings/api"
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
