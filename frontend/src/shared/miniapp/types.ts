export type Lang = "ru" | "kk" | "en"

export type TabKey = "home" | "meetings" | "checker" | "profile"

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

export type { Meeting, MeetingDraft } from "@/entities/meeting/types"
export type { Employee } from "@/entities/employee/types"
