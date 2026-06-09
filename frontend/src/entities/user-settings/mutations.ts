import { useMutation, useQueryClient } from "@tanstack/react-query"
import { patchUserSettings } from "./write-api"
import { userSettingsKeys } from "./queries"
import type { UserSettings } from "./types"

export function useUpdateReminderMinutes() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (reminderMinutes: number[]) => patchUserSettings(reminderMinutes),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: userSettingsKeys.current() })
      const prev = qc.getQueryData<UserSettings>(userSettingsKeys.current())
      qc.setQueryData<UserSettings>(userSettingsKeys.current(), {
        reminderMinutes: [...next].sort((a, b) => a - b),
      })
      return { prev }
    },
    onError: (_err, _next, ctx) => {
      if (ctx?.prev) qc.setQueryData(userSettingsKeys.current(), ctx.prev)
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: userSettingsKeys.current() })
    },
  })
}
