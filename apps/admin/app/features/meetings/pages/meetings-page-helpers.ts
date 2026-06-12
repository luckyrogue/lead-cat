import type {
  CreateMeetingInput,
  Meeting,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"
import type {
  MeetingFormDefaults,
  MeetingFormValues,
} from "~/features/meetings/components/meeting-form"

function splitParticipants(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((email) => email.trim())
    .filter(Boolean)
}

export function toCreateInput(values: MeetingFormValues): CreateMeetingInput {
  return {
    dept: values.dept,
    type: values.type,
    host: values.host || undefined,
    date: values.date,
    start: values.start,
    end: values.end,
    recurrence: values.recurrence,
    desc: values.desc || undefined,
    participants: splitParticipants(values.participants),
    recurrence_until:
      values.recurrence === "once" ? undefined : values.recurrence_until,
    recurrence_days:
      values.recurrence === "custom" ? values.recurrence_days : undefined,
  }
}

export function toUpdateInput(
  values: MeetingFormValues,
  scope: MeetingScope
): UpdateMeetingInput {
  return {
    dept: values.dept,
    type: values.type,
    host: values.host || undefined,
    date: scope === "whole" ? undefined : values.date,
    start: values.start,
    end: values.end,
    desc: values.desc,
  }
}

function localDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ""
  }
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function localTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ""
  }
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function editDefaults(meeting: Meeting): MeetingFormDefaults {
  return {
    dept: meeting.dept,
    type: meeting.type,
    host: meeting.host,
    date: localDate(meeting.starts_at),
    start: localTime(meeting.starts_at),
    end: localTime(meeting.ends_at),
    desc: meeting.description,
  }
}
