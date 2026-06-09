// Must stay in sync with backend/internal/platform/botsettings/settings.go Intervals.
export const REMINDER_INTERVALS = [
  { value: 10, labelKey: "rem10m" as const },
  { value: 15, labelKey: "rem15m" as const },
  { value: 30, labelKey: "rem30m" as const },
  { value: 60, labelKey: "rem1h" as const },
  { value: 120, labelKey: "rem2h" as const },
  { value: 1440, labelKey: "rem1d" as const },
] as const

export type ReminderMinute = (typeof REMINDER_INTERVALS)[number]["value"]
