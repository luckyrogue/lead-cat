import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Label,
} from "@leadcat/ui"

import { useMeSettings, useUpdateMeSettings } from "~/entities/me/queries"
import { useT } from "~/shared/i18n/context"
import { toastError, toastSuccess } from "~/shared/lib/toast"

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

  function handleTimezoneChange(e: React.ChangeEvent<HTMLSelectElement>) {
    updateSettings.mutate(
      { timezone: e.target.value },
      {
        onSuccess: () => toastSuccess(t("settings.toast.timezoneSaved")),
        onError: (error) =>
          toastError(error, t("settings.toast.timezoneFailed")),
      }
    )
  }

  function handleLanguageChange(e: React.ChangeEvent<HTMLSelectElement>) {
    updateSettings.mutate(
      { language: e.target.value },
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
            <select
              id="timezone-select"
              value={isPending ? "" : (settings?.timezone ?? "")}
              disabled={isPending || updateSettings.isPending}
              onChange={handleTimezoneChange}
              className="flex h-9 w-full max-w-xs rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {TIMEZONE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {"translatable" in opt
                    ? t("settings.timezoneBrowserDefault")
                    : opt.label}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="language-select">
              {t("settings.languageLabel")}
            </Label>
            <select
              id="language-select"
              value={isPending ? "" : (settings?.language ?? "")}
              disabled={isPending || updateSettings.isPending}
              onChange={handleLanguageChange}
              className="flex h-9 w-full max-w-xs rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {LANGUAGE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {"translatable" in opt
                    ? t("settings.languageDefault")
                    : opt.label}
                </option>
              ))}
            </select>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
