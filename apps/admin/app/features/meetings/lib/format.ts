import type { Meeting } from "~/entities/meeting/types"

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

export function formatTimeRange(meeting: Meeting, timeZone?: string): string {
  const start = new Date(meeting.starts_at)
  const end = new Date(meeting.ends_at)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return "—"
  }
  const time = (d: Date) =>
    d.toLocaleTimeString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      ...(timeZone ? { timeZone } : {}),
    })
  return `${time(start)} – ${time(end)}`
}
