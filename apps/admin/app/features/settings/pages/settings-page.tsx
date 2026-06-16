import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Label,
} from "@leadcat/ui"

import { useMeSettings, useUpdateMeSettings } from "~/entities/me/queries"
import { toastError, toastSuccess } from "~/shared/lib/toast"

const TIMEZONE_OPTIONS = [
  { value: "", label: "Browser default" },
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
  { value: "", label: "Default" },
  { value: "ru", label: "Русский" },
  { value: "en", label: "English" },
  { value: "kk", label: "Қазақша" },
]

export function SettingsPage() {
  const { data: settings, isPending } = useMeSettings()
  const updateSettings = useUpdateMeSettings()

  function handleTimezoneChange(e: React.ChangeEvent<HTMLSelectElement>) {
    updateSettings.mutate(
      { timezone: e.target.value },
      {
        onSuccess: () => toastSuccess("Timezone saved."),
        onError: (error) => toastError(error, "Could not save timezone."),
      }
    )
  }

  function handleLanguageChange(e: React.ChangeEvent<HTMLSelectElement>) {
    updateSettings.mutate(
      { language: e.target.value },
      {
        onSuccess: () => toastSuccess("Language saved."),
        onError: (error) => toastError(error, "Could not save language."),
      }
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-foreground">Settings</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Manage your personal preferences.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
          <CardDescription>
            Choose how dates and times are displayed across the admin panel.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="space-y-1.5">
            <Label htmlFor="timezone-select">Timezone</Label>
            <select
              id="timezone-select"
              value={isPending ? "" : (settings?.timezone ?? "")}
              disabled={isPending || updateSettings.isPending}
              onChange={handleTimezoneChange}
              className="flex h-9 w-full max-w-xs rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            >
              {TIMEZONE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="language-select">Language</Label>
            <select
              id="language-select"
              value={isPending ? "" : (settings?.language ?? "")}
              disabled={isPending || updateSettings.isPending}
              onChange={handleLanguageChange}
              className="flex h-9 w-full max-w-xs rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs transition-colors focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
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
    </div>
  )
}
