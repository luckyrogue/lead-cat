import { api } from "@/shared/api/client"
import { fmtDate } from "./meeting-utils"
import type { Employee, FreeSlot, Lang, Meeting } from "./types"

export type Scope = "upcoming" | "past" | "all"

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
  const res = await api.get<{ meetings: MeetingDTO[] }>("/tma/meetings", {
    params: { scope },
  })
  return res.data.meetings.map(toMeeting)
}

export async function fetchColleagueSchedule(
  email: string,
  scope: Scope
): Promise<Meeting[]> {
  const res = await api.get<{ meetings: MeetingDTO[] }>("/tma/schedule", {
    params: { email, scope },
  })
  return res.data.meetings.map(toMeeting)
}

export async function searchEmployees(q: string): Promise<Employee[]> {
  const res = await api.get<{ employees: EmployeeDTO[] }>("/tma/employees", {
    params: { q },
  })
  return res.data.employees.map((e) => ({
    id: e.id,
    name: e.name,
    email: e.email,
    dept: e.dept,
    tg: e.tg,
  }))
}

export type FreeSlotsParams = {
  participants: string[]
  from: string // YYYY-MM-DD
  to: string // YYYY-MM-DD
  durationMins: number
}

export async function fetchFreeSlots(
  params: FreeSlotsParams,
  lang: Lang
): Promise<FreeSlot[]> {
  const res = await api.post<{ slots: FreeSlotDTO[] }>("/tma/free-slots", {
    participants: params.participants,
    from: params.from,
    to: params.to,
    duration_mins: params.durationMins,
  })
  return res.data.slots.map((s) => ({
    day: fmtDate(s.iso, lang),
    iso: s.iso,
    start: s.start,
    end: s.end,
    mins: s.mins,
  }))
}
