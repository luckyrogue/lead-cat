import { Card, CardContent, toast } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useEffect, useState } from "react"

import { ErrorState, LoadingState } from "~/components/states"
import { LANGUAGE_OPTIONS, TIMEZONE_OPTIONS } from "~/entities/settings/api"
import { settingsQuery, useUpdatePrefs } from "~/entities/settings/queries"

export function PreferencesSettings() {
  const settings = useQuery(settingsQuery())
  const update = useUpdatePrefs()
  const [timezone, setTimezone] = useState("")
  const [language, setLanguage] = useState("")

  useEffect(() => {
    if (settings.data) {
      setTimezone(settings.data.timezone ?? "")
      setLanguage(settings.data.language ?? "")
    }
  }, [settings.data])

  function handleTimezoneChange(value: string) {
    const prev = timezone
    setTimezone(value)
    update.mutate(
      { timezone: value },
      {
        onError: () => {
          toast.error("Couldn't save preferences")
          setTimezone(prev)
        },
      }
    )
  }

  function handleLanguageChange(value: string) {
    const prev = language
    setLanguage(value)
    update.mutate(
      { language: value },
      {
        onError: () => {
          toast.error("Couldn't save preferences")
          setLanguage(prev)
        },
      }
    )
  }

  if (settings.isLoading) {
    return <LoadingState />
  }
  if (settings.isError) {
    return (
      <ErrorState
        title="Couldn't load settings"
        onRetry={() => settings.refetch()}
      />
    )
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm font-semibold text-foreground">
          Timezone &amp; language
        </p>
        <div className="flex flex-col gap-2">
          <label className="text-xs text-muted-foreground">Timezone</label>
          <select
            className="h-10 rounded-[var(--radius)] border border-border bg-background px-3 text-sm"
            value={timezone}
            disabled={update.isPending}
            onChange={(e) => handleTimezoneChange(e.target.value)}
          >
            {TIMEZONE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-2">
          <label className="text-xs text-muted-foreground">Language</label>
          <select
            className="h-10 rounded-[var(--radius)] border border-border bg-background px-3 text-sm"
            value={language}
            disabled={update.isPending}
            onChange={(e) => handleLanguageChange(e.target.value)}
          >
            {LANGUAGE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </CardContent>
    </Card>
  )
}
