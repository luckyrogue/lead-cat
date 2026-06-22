import { Card, CardContent, cn, toastError } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"

import { ErrorState, LoadingState } from "~/components/states"
import { getReminderOptions } from "~/entities/settings/api"
import {
  settingsQuery,
  useUpdateReminderMinutes,
} from "~/entities/settings/queries"
import { useT } from "~/shared/i18n/context"

export function ReminderSettings() {
  const t = useT()
  const reminderOptions = useMemo(() => getReminderOptions(t), [t])
  const settings = useQuery(settingsQuery())
  const update = useUpdateReminderMinutes()
  const [selected, setSelected] = useState<number[]>([])

  useEffect(() => {
    if (settings.data) {
      setSelected(settings.data.reminder_minutes)
    }
  }, [settings.data])

  function toggle(minutes: number) {
    const next = selected.includes(minutes)
      ? selected.filter((m) => m !== minutes)
      : [...selected, minutes].sort((a, b) => a - b)
    setSelected(next)
    update.mutate(next, {
      onError: (error) => {
        toastError(error, t, "profile.reminder.toastError")
        setSelected(settings.data?.reminder_minutes ?? [])
      },
    })
  }

  if (settings.isLoading) {
    return <LoadingState />
  }
  if (settings.isError) {
    return (
      <ErrorState
        title={t("profile.reminder.toastLoadError")}
        onRetry={() => settings.refetch()}
      />
    )
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm font-semibold text-foreground">
          {t("profile.reminder.title")}
        </p>
        <div className="flex flex-wrap gap-2">
          {reminderOptions.map((option) => {
            const active = selected.includes(option.minutes)
            return (
              <button
                key={option.minutes}
                type="button"
                onClick={() => toggle(option.minutes)}
                disabled={update.isPending}
                className={cn(
                  "rounded-full border px-3 py-1.5 text-sm font-medium transition-colors disabled:opacity-60",
                  active
                    ? "border-primary bg-primary/10 text-primary"
                    : "border-border/70 text-muted-foreground"
                )}
              >
                {option.label}
              </button>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}
