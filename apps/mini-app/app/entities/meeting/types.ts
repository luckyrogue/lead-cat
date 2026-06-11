export type Meeting = {
  id: string
  type: string
  dept: string
  host: string
  date: string
  start: string
  end: string
  rec: string
  organizer: string
  participants: string[]
  desc: string
  meet_link: string
  status: string
}

export type MeetingRecurrence = "once" | "daily" | "weekly" | "monthly"

export type CreateMeetingInput = {
  dept: string
  type: string
  host: string
  date: string
  start: string
  end: string
  recurrence: string
  recurrence_until?: string
  desc: string
  participants: string[]
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

export type Conflict = {
  email: string
  name: string
  title: string
  start: string
  end: string
}

export type OccurrenceConflicts = {
  date: string
  start: string
  end: string
  conflicts: Conflict[]
}

export type FreeSlot = {
  iso: string
  start: string
  end: string
  mins: number
}

export type MeetingMutationScope = "this" | "whole"

export function isSeriesMeeting(meeting: Pick<Meeting, "rec">): boolean {
  return meeting.rec !== "" && meeting.rec !== "once"
}
