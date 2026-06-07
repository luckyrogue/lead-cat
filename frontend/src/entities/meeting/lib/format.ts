import { MEETING_TYPES, RECURRENCE } from "@/entities/meeting/constants"
import type { Meeting, MeetingDraft } from "@/entities/meeting/types"
import type { Employee } from "@/entities/employee/types"
import { emailsToPeople } from "@/entities/employee/fixtures"
export type Lang = "ru" | "kk" | "en"

export type PartWordKey = "part1" | "part24" | "participants"

export function buildTitle(
  m: Pick<Meeting, "type" | "dept" | "host" | "date" | "rec">
): string {
  const t = MEETING_TYPES.find((x) => x.key === m.type)
  const d = m.date ? m.date.split("-").reverse().join(".") : "—"
  const recObj = RECURRENCE.find((r) => r.key === m.rec)
  const hostParts = m.host ? m.host.split(" ") : []
  const hostShort =
    hostParts.length > 0
      ? `${hostParts[0]} ${hostParts[1] ? `${hostParts[1][0]}.` : ""}`
      : ""
  const parts = [m.dept, t ? t.label : m.type, hostShort, d]
  if (recObj?.short) parts.push(recObj.short)
  return parts.join(" · ")
}

export function fmtDate(iso: string, lang: Lang): string {
  const [y, m, d] = iso.split("-").map(Number)
  const dt = new Date(y, m - 1, d)
  const dow = ["вс", "пн", "вт", "ср", "чт", "пт", "сб"][dt.getDay()]
  const dowEn = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"][dt.getDay()]
  const dd = String(d).padStart(2, "0")
  const mm = String(m).padStart(2, "0")
  return lang === "en" ? `${dowEn} ${dd}.${mm}` : `${dd}.${mm}, ${dow}`
}

export function partWord(n: number, t: (k: PartWordKey) => string): string {
  const last = n % 10
  const last2 = n % 100
  if (last === 1 && last2 !== 11) return t("part1")
  if (last >= 2 && last <= 4 && (last2 < 10 || last2 >= 20)) return t("part24")
  return t("participants")
}

export function meetCount(n: number, lang: Lang): string {
  if (lang === "en") return `${n} ${n === 1 ? "meeting" : "meetings"}`
  if (lang === "kk") return `${n} кездесу`
  const last = n % 10
  const last2 = n % 100
  let w = "встреч"
  if (last === 1 && last2 !== 11) w = "встреча"
  else if (last >= 2 && last <= 4 && (last2 < 10 || last2 >= 20)) w = "встречи"
  return `${n} ${w}`
}

export function detailToDraft(m: Meeting): Partial<MeetingDraft> {
  const [h, mn] = m.start.split(":").map(Number)
  const [eh, emn] = m.end.split(":").map(Number)
  const dur = eh * 60 + emn - (h * 60 + mn)
  return {
    dept: m.dept,
    type: m.type,
    host: m.host,
    date: m.date,
    start: m.start,
    dur: dur > 0 ? dur : 30,
    rec: m.rec,
    recDays: m.recDays ?? [],
    participants: emailsToPeople(m.participants),
    desc: m.desc ?? "",
  }
}

export function draftToMeeting(
  draft: MeetingDraft & { end: string },
  id: string,
  organizer: string
): Meeting {
  return {
    id,
    type: draft.type,
    dept: draft.dept,
    host: draft.host,
    date: draft.date,
    start: draft.start,
    end: draft.end,
    rec: draft.rec,
    recDays: draft.recDays,
    organizer,
    participants: draft.participants.map((p: Employee) => p.email),
    desc: draft.desc,
  }
}
