import { z } from "zod"
import { minutesToTime, timeToMinutes } from "@leadcat/ui"

import type {
  BookingEventType,
  EventTypeInput,
} from "~/entities/booking-event-type/types"

export const DURATION_OPTIONS = [15, 30, 45, 60] as const

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
    survey_id: z.string().nullable(),
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
    survey_id: et.survey_id ?? null,
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
    survey_id: values.survey_id ?? null,
  }
}
