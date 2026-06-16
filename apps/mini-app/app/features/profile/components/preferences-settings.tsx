import {
  Card,
  CardContent,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useEffect, useState } from "react"

import { ErrorState, LoadingState } from "~/components/states"
import { LANGUAGE_OPTIONS, TIMEZONE_OPTIONS } from "~/entities/settings/api"
import { settingsQuery, useUpdatePrefs } from "~/entities/settings/queries"
import { useT } from "~/shared/i18n/context"

// Radix Select forbids an empty-string value, so the "default" option rides a
// sentinel that maps back to "" at the data boundary.
const DEFAULT = "__default__"

export function PreferencesSettings() {
  const t = useT()
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
          toast.error(t("profile.preferences.toastError"))
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
          toast.error(t("profile.preferences.toastError"))
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
        title={t("profile.preferences.toastLoadError")}
        onRetry={() => settings.refetch()}
      />
    )
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm font-semibold text-foreground">
          {t("profile.preferences.title")}
        </p>
        <div className="flex flex-col gap-2">
          <label className="text-xs text-muted-foreground">
            {t("profile.preferences.timezoneLabel")}
          </label>
          <Select
            value={timezone === "" ? DEFAULT : timezone}
            disabled={update.isPending}
            onValueChange={(v) => handleTimezoneChange(v === DEFAULT ? "" : v)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TIMEZONE_OPTIONS.map((opt) => (
                <SelectItem
                  key={opt.value || "default"}
                  value={opt.value === "" ? DEFAULT : opt.value}
                >
                  {opt.value === ""
                    ? t("profile.preferences.tzDefault")
                    : opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-2">
          <label className="text-xs text-muted-foreground">
            {t("profile.preferences.languageLabel")}
          </label>
          <Select
            value={language === "" ? DEFAULT : language}
            disabled={update.isPending}
            onValueChange={(v) => handleLanguageChange(v === DEFAULT ? "" : v)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {LANGUAGE_OPTIONS.map((opt) => (
                <SelectItem
                  key={opt.value || "default"}
                  value={opt.value === "" ? DEFAULT : opt.value}
                >
                  {opt.value === ""
                    ? t("profile.preferences.langDefault")
                    : opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </CardContent>
    </Card>
  )
}
