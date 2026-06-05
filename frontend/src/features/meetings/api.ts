import { apiFetch } from "@/shared/api/client"
import type { TmaMeetingsScope } from "@/shared/api/types"
import type { Employee } from "@/entities/employee/types"
import type { Meeting } from "@/entities/meeting/types"
import { fmtDate } from "@/shared/tma/meeting-utils"
import type { Lang } from "@/shared/tma/types"

export type Scope = TmaMeetingsScope

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
}

type EmployeeDTO = {
  id: string
  name: string
  email: string
  dept: string
  tg: boolean
}

type FreeSlotDTO = { iso: string; start: string; end: string; mins: number }

export type FreeSlot = {
  day: string
  iso: string
  start: string
  end: string
  mins: number
}

function toMeeting(d: MeetingDTO): Meeting {
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
  }
}

export async function fetchMyMeetings(scope: Scope): Promise<Meeting[]> {
  const data = await apiFetch<{ meetings: MeetingDTO[] }>("/tma/meetings", {
    params: { scope },
  })
  return data.meetings.map(toMeeting)
}

export async function fetchColleagueSchedule(
  email: string,
  scope: Scope
): Promise<Meeting[]> {
  const data = await apiFetch<{ meetings: MeetingDTO[] }>("/tma/schedule", {
    params: { email, scope },
  })
  return data.meetings.map(toMeeting)
}

export async function searchEmployees(
  q: string,
  signal?: AbortSignal
): Promise<Employee[]> {
  const data = await apiFetch<{ employees: EmployeeDTO[] }>("/tma/employees", {
    params: { q },
    signal,
  })
  return data.employees.map((e) => ({
    id: e.id,
    name: e.name,
    email: e.email,
    dept: e.dept,
    tg: e.tg,
  }))
}

export type FreeSlotsParams = {
  participants: string[]
  from: string
  to: string
  durationMins: number
}

export async function fetchFreeSlots(
  params: FreeSlotsParams,
  lang: Lang
): Promise<FreeSlot[]> {
  const data = await apiFetch<{ slots: FreeSlotDTO[] }>("/tma/free-slots", {
    method: "POST",
    body: {
      participants: params.participants,
      from: params.from,
      to: params.to,
      duration_mins: params.durationMins,
    },
  })
  return data.slots.map((s) => ({
    day: fmtDate(s.iso, lang),
    iso: s.iso,
    start: s.start,
    end: s.end,
    mins: s.mins,
  }))
}
