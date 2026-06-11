export type MeetingParticipant = {
  employee_id?: string | null
  email: string
}

export type Meeting = {
  id: string
  organization_id: string
  organizer_user_id?: string | null
  dept: string
  type: string
  host: string
  starts_at: string
  ends_at: string
  recurrence: string
  name: string
  description: string
  google_event_id: string
  meet_link: string
  status: string
  series_id?: string | null
  recurrence_until?: string | null
  recurrence_days?: number[] | null
  participants: MeetingParticipant[]
}

export type MeetingRecurrence =
  | "once"
  | "daily"
  | "weekly"
  | "monthly"
  | "custom"

export type MeetingScope = "this" | "whole"

export function isSeries(meeting: Meeting): boolean {
  return meeting.recurrence !== "once" || Boolean(meeting.series_id)
}

export type CreateMeetingInput = {
  dept: string
  type: string
  host?: string
  date: string
  start: string
  end: string
  recurrence: MeetingRecurrence
  desc?: string
  participants?: string[]
  recurrence_until?: string
  recurrence_days?: number[]
}

export type UpdateMeetingInput = {
  dept?: string
  type?: string
  host?: string
  date?: string
  start?: string
  end?: string
  desc?: string
}
