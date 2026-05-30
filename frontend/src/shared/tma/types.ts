export type Lang = "ru" | "kk" | "en"

export type TabKey = "home" | "meetings" | "checker" | "auto" | "profile"

export type Employee = {
  id: string
  name: string
  email: string
  dept: string
  tg: boolean
  role?: "admin"
}

export type Meeting = {
  id: string
  type: string
  dept: string
  host: string
  date: string
  start: string
  end: string
  rec: string
  recDays?: number[]
  organizer: string
  participants: string[]
  desc?: string
}

export type MeetingDraft = {
  dept: string
  type: string
  host: string
  date: string
  start: string
  dur: number
  rec: string
  recDays: number[]
  participants: Employee[]
  desc: string
  end?: string
}

export type Scenario = {
  id: string
  name: string
  enabled: boolean
  trigger: { hour: number; minute: number; days: number[] }
  actions: string[]
  note: string
}

export type FreeSlot = {
  day: string
  iso: string
  start: string
  end: string
  mins: number
}

export type CatPalette = {
  dark: boolean
  accent: string
  accentText: string
  accentSoft: string
  accentLine: string
  bg: string
  bg2: string
  card: string
  cardAlt: string
  text: string
  muted: string
  faint: string
  border: string
  borderStrong: string
  tgBar: string
  tgBarText: string
  shadow: string
  shadowSm: string
  pattern: string
  danger: string
  dangerSoft: string
  ok: string
  okSoft: string
}

export type OverlayState =
  | { type: "create"; initial?: Partial<MeetingDraft> }
  | { type: "colleague" }
  | { type: "admin" }
  | null
