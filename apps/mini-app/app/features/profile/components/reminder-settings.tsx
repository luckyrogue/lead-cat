import { Card, CardContent, cn, toast } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useEffect, useState } from "react"

import { ErrorState, LoadingState } from "~/components/states"
import { REMINDER_OPTIONS } from "~/entities/settings/api"
import { settingsQuery, useUpdateReminderMinutes } from "~/entities/settings/queries"

export function ReminderSettings() {
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
      onError: () => {
        toast.error("Couldn't save reminders")
        setSelected(settings.data?.reminder_minutes ?? [])
      },
    })
  }

  if (settings.isLoading) {
    return <LoadingState />
  }
  if (settings.isError) {
    return <ErrorState title="Couldn't load settings" onRetry={() => settings.refetch()} />
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm font-semibold text-foreground">Reminders before a meeting</p>
        <div className="flex flex-wrap gap-2">
          {REMINDER_OPTIONS.map((option) => {
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
