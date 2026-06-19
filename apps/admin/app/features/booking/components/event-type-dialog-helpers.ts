import { z } from "zod"

import type { BookingEventType, EventTypeInput } from "~/entities/booking-event-type/types"

export const DURATION_OPTIONS = [15, 30, 45, 60] as const

export const WEEKDAYS = [
  { value: 1, key: "1" },
  { value: 2, key: "2" },
  { value: 3, key: "3" },
  { value: 4, key: "4" },
  { value: 5, key: "5" },
  { value: 6, key: "6" },
  { value: 7, key: "7" },
] as const

export const TIMEZONE_OPTIONS = [
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

export function minutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
    .toString()
    .padStart(2, "0")
  const m = (minutes % 60).toString().padStart(2, "0")
  return `${h}:${m}`
}

export function timeToMinutes(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return "UTC"
  }
}

export const schema = z
  .object({
    title: z.string().min(1, "booking.errors.titleRequired"),
    description: z.string(),
    duration_mins: z.number().int().positive(),
    timezone: z.string().min(1, "booking.errors.timezoneRequired"),
    avail_weekdays: z
      .array(z.number().int().min(1).max(7))
      .min(1, "booking.errors.weekdayRequired"),
    avail_start_time: z.string().min(1, "booking.errors.startRequired"),
    avail_end_time: z.string().min(1, "booking.errors.endRequired"),
    active: z.boolean(),
  })
  .refine((v) => v.avail_end_time > v.avail_start_time, {
    path: ["avail_end_time"],
    message: "booking.errors.endAfterStart",
  })

export type FormValues = z.infer<typeof schema>

export function toFormValues(et: BookingEventType): FormValues {
  return {
    title: et.title,
    description: et.description,
    duration_mins: et.duration_mins,
    timezone: et.timezone,
    avail_weekdays: et.avail_weekdays,
    avail_start_time: minutesToTime(et.avail_start_minute),
    avail_end_time: minutesToTime(et.avail_end_minute),
    active: et.active,
  }
}

export function toInput(values: FormValues): EventTypeInput {
  return {
    title: values.title,
    description: values.description,
    duration_mins: values.duration_mins,
    timezone: values.timezone,
    avail_weekdays: values.avail_weekdays,
    avail_start_minute: timeToMinutes(values.avail_start_time),
    avail_end_minute: timeToMinutes(values.avail_end_time),
    active: values.active,
  }
}

export function toggleWeekday(days: number[], day: number): number[] {
  return days.includes(day) ? days.filter((d) => d !== day) : [...days, day]
}
