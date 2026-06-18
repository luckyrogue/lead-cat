import { format, parse } from "date-fns"
import { enUS as dateFnsEnUS, ru as dateFnsRu } from "date-fns/locale"
import type { Locale as DateFnsLocale } from "date-fns"
import { enUS, ru } from "react-day-picker/locale"
import type { Locale as DayPickerLocale } from "react-day-picker"

const ISO_DATE = "yyyy-MM-dd"

export function parseIsoDate(value?: string): Date | undefined {
  if (!value) {
    return undefined
  }
  const parsed = parse(value, ISO_DATE, new Date())
  return Number.isNaN(parsed.getTime()) ? undefined : parsed
}

export function formatIsoDate(date: Date): string {
  return format(date, ISO_DATE)
}

export function parseTimeValue(value?: string): { hour: number; minute: number } {
  if (!value) {
    return { hour: 0, minute: 0 }
  }
  const [hourRaw, minuteRaw] = value.split(":")
  const hour = Number(hourRaw)
  const minute = Number(minuteRaw)
  if (Number.isNaN(hour) || Number.isNaN(minute)) {
    return { hour: 0, minute: 0 }
  }
  return { hour, minute }
}

export function formatTimeValue(hour: number, minute: number): string {
  return `${String(hour).padStart(2, "0")}:${String(minute).padStart(2, "0")}`
}

export function dateFnsLocaleFromCode(code?: string): DateFnsLocale {
  if (code === "ru" || code === "kk") {
    return dateFnsRu
  }
  return dateFnsEnUS
}

export function dayPickerLocaleFromCode(code?: string): DayPickerLocale {
  if (code === "ru" || code === "kk") {
    return ru
  }
  return enUS
}

export function formatDateLabel(
  iso: string,
  locale: DateFnsLocale = dateFnsEnUS,
  pattern = "PPP",
): string {
  const date = parseIsoDate(iso)
  if (!date) {
    return iso
  }
  return format(date, pattern, { locale })
}

export function buildTimeOptions(step = 5): string[] {
  const options: string[] = []
  for (let hour = 0; hour < 24; hour += 1) {
    for (let minute = 0; minute < 60; minute += step) {
      options.push(formatTimeValue(hour, minute))
    }
  }
  return options
}

export const DEFAULT_MEETING_DURATION_MIN = 60

export function timeToMinutes(time: string): number {
  const { hour, minute } = parseTimeValue(time)
  return hour * 60 + minute
}

export function diffMinutes(start: string, end: string): number {
  if (!start || !end) {
    return DEFAULT_MEETING_DURATION_MIN
  }
  const delta = timeToMinutes(end) - timeToMinutes(start)
  return delta > 0 ? delta : DEFAULT_MEETING_DURATION_MIN
}

export function addMinutesToTime(time: string, minutes: number): string {
  const total = timeToMinutes(time) + minutes
  const normalized = ((total % (24 * 60)) + 24 * 60) % (24 * 60)
  const hh = Math.floor(normalized / 60)
  const mm = normalized % 60
  return formatTimeValue(hh, mm)
}

export function todayIso(): string {
  return formatIsoDate(new Date())
}

export type IsoDateRange = {
  from: string
  to: string
}

export function formatDateRangeLabel(
  range: IsoDateRange,
  localeCode?: string,
  pattern = "d MMM yyyy",
): string {
  const locale = dateFnsLocaleFromCode(localeCode)
  const { from, to } = range
  if (from && to) {
    if (from === to) {
      return formatDateLabel(from, locale, pattern)
    }
    return `${formatDateLabel(from, locale, pattern)} – ${formatDateLabel(to, locale, pattern)}`
  }
  if (from) {
    return formatDateLabel(from, locale, pattern)
  }
  if (to) {
    return formatDateLabel(to, locale, pattern)
  }
  return ""
}
