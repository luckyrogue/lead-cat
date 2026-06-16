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
import { useT } from "~/shared/i18n/context"
import { toastError, toastSuccess } from "~/shared/lib/toast"

// Radix Select forbids an empty-string value, so the "default" option rides a
// sentinel that maps back to "" at the data boundary.
const DEFAULT = "__default__"

const TIMEZONE_OPTIONS = [
  { value: "", translatable: true },
  { value: "Asia/Almaty", label: "Almaty (UTC+5)" },
  { value: "Asia/Tashkent", label: "Tashkent (UTC+5)" },
  { value: "Asia/Bishkek", label: "Bishkek (UTC+6)" },
  { value: "Europe/Moscow", label: "Moscow (UTC+3)" },
  { value: "Europe/Kyiv", label: "Kyiv (UTC+2/3)" },
  { value: "Europe/London", label: "London (UTC+0/1)" },
  { value: "Asia/Dubai", label: "Dubai (UTC+4)" },
  { value: "Asia/Istanbul", label: "Istanbul (UTC+3)" },
  { value: "America/New_York", label: "New York (UTC-5/4)" },
  { value: "UTC", label: "UTC" },
]

const LANGUAGE_OPTIONS = [
  { value: "", translatable: true },
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
  { value: "kk", label: "Қазақша" },
]

export function SettingsPage() {
  const t = useT()
  const { data: settings, isPending } = useMeSettings()
  const updateSettings = useUpdateMeSettings()

  function handleTimezoneChange(value: string) {
    updateSettings.mutate(
      { timezone: value },
      {
        onSuccess: () => toastSuccess(t("settings.toast.timezoneSaved")),
        onError: (error) =>
          toastError(error, t("settings.toast.timezoneFailed")),
      }
    )
  }

  function handleLanguageChange(value: string) {
    updateSettings.mutate(
      { language: value },
      {
        onSuccess: () => toastSuccess(t("settings.toast.languageSaved")),
        onError: (error) =>
          toastError(error, t("settings.toast.languageFailed")),
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
                {TIMEZONE_OPTIONS.map((opt) => (
                  <SelectItem
                    key={opt.value}
                    value={opt.value === "" ? DEFAULT : opt.value}
                  >
                    {"translatable" in opt
                      ? t("settings.timezoneBrowserDefault")
                      : opt.label}
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
                    key={opt.value}
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
