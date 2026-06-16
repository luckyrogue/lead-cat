import type { Meeting } from "~/entities/meeting/types"

type FormatOptions = {
  timeZone?: string
  locale?: string
}

export function formatMeetingDate(
  value: string,
  { timeZone, locale }: FormatOptions = {},
): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "—"
  }
  return date.toLocaleDateString(locale, {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
    ...(timeZone ? { timeZone } : {}),
  })
}

export function formatDateTime(value: string, timeZone?: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "—"
  }
  return date.toLocaleString(undefined, {
    weekday: "short",
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
    ...(timeZone ? { timeZone } : {}),
  })
}

export function formatTimeRange(
  meeting: Meeting,
  { timeZone, locale }: FormatOptions = {},
): string {
  const start = new Date(meeting.starts_at)
  const end = new Date(meeting.ends_at)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return "—"
  }
  const time = (d: Date) =>
    d.toLocaleTimeString(locale, {
      hour: "2-digit",
      minute: "2-digit",
      ...(timeZone ? { timeZone } : {}),
    })
  return `${time(start)} – ${time(end)}`
}

export function meetingTitle(meeting: Meeting, fallback: string): string {
  if (meeting.type.trim()) {
    return meeting.type.trim()
  }
  const fromName = meeting.name.split("|").map((part) => part.trim())[1]
  return fromName || fallback
}
