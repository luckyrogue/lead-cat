import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import { useMeSettings, useUpdateMeSettings } from "~/entities/me/queries"
import { CalendarConnectionsCard } from "~/features/calendar-connections/components/calendar-connections-card"
import { resolveLocale, useT } from "~/shared/i18n/context"
import { writeLocalePreference } from "~/shared/i18n/locale-preference"
import { getTimezoneOptionsWithEmpty } from "~/shared/lib/timezone-options"
import { toastError, toastSuccess } from "~/shared/lib/toast"

const DEFAULT = "__default__"

const LANGUAGE_OPTIONS = [
  { value: "", translatable: true },
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
  { value: "kk", label: "Қазақша" },
]

export function SettingsPage() {
  const t = useT()
  const timezoneOptions = getTimezoneOptionsWithEmpty(
    t,
    "settings.timezoneBrowserDefault"
  )
  const { data: settings, isPending } = useMeSettings()
  const updateSettings = useUpdateMeSettings()

  function handleTimezoneChange(value: string) {
    updateSettings.mutate(
      { timezone: value },
      {
        onSuccess: () => toastSuccess(t("settings.toast.timezoneSaved")),
        onError: (error) =>
          toastError(error, t, "settings.toast.timezoneFailed"),
      }
    )
  }

  function handleLanguageChange(value: string) {
    updateSettings.mutate(
      { language: value },
      {
        onSuccess: () => {
          if (value) {
            writeLocalePreference(resolveLocale(value))
          }
          toastSuccess(t("settings.toast.languageSaved"))
        },
        onError: (error) =>
          toastError(error, t, "settings.toast.languageFailed"),
      }
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-foreground">
          {t("settings.pageTitle")}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("settings.pageDescription")}
        </p>
      </div>

      <CalendarConnectionsCard />

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.cardTitle")}</CardTitle>
          <CardDescription>{t("settings.cardDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="space-y-1.5">
            <Label htmlFor="timezone-select">
              {t("settings.timezoneLabel")}
            </Label>
            <Select
              value={settings?.timezone || DEFAULT}
              disabled={isPending || updateSettings.isPending}
              onValueChange={(v) =>
                handleTimezoneChange(v === DEFAULT ? "" : v)
              }
            >
              <SelectTrigger id="timezone-select" className="max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {timezoneOptions.map((opt) => (
                  <SelectItem
                    key={opt.value || "default"}
                    value={opt.value === "" ? DEFAULT : opt.value}
                  >
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="language-select">
              {t("settings.languageLabel")}
            </Label>
            <Select
              value={settings?.language || DEFAULT}
              disabled={isPending || updateSettings.isPending}
              onValueChange={(v) =>
                handleLanguageChange(v === DEFAULT ? "" : v)
              }
            >
              <SelectTrigger id="language-select" className="max-w-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LANGUAGE_OPTIONS.map((opt) => (
                  <SelectItem
                    key={opt.value || "default"}
                    value={opt.value === "" ? DEFAULT : opt.value}
                  >
                    {"translatable" in opt
                      ? t("settings.languageDefault")
                      : opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
