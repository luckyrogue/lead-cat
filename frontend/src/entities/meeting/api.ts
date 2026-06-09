import { apiFetch } from "@/shared/api/client"
import type { MiniAppMeetingsScope } from "@/shared/api/types"
import type { Meeting } from "@/entities/meeting/types"

export type Scope = MiniAppMeetingsScope

type MeetingDTO = {
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
  series_id?: string
  recurrence_until?: string
}

export function toMeeting(d: MeetingDTO): Meeting {
  return {
    id: d.id,
    type: d.type,
    dept: d.dept,
    host: d.host,
    date: d.date,
    start: d.start,
    end: d.end,
    rec: d.rec,
    organizer: d.organizer,
    participants: d.participants,
    desc: d.desc,
    seriesId: d.series_id,
    recurrenceUntil: d.recurrence_until,
  }
}

export async function fetchMyMeetings(scope: Scope): Promise<Meeting[]> {
  const data = await apiFetch<{ meetings: MeetingDTO[] }>("/miniapp/meetings", {
    params: { scope },
  })
  return data.meetings.map(toMeeting)
}

export async function fetchColleagueSchedule(
  email: string,
  scope: Scope
): Promise<Meeting[]> {
  const data = await apiFetch<{ meetings: MeetingDTO[] }>("/miniapp/schedule", {
    params: { email, scope },
  })
  return data.meetings.map(toMeeting)
}
