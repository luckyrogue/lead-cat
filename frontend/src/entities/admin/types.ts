export type WorkspaceStatus = {
  id: string
  name: string
  tz: string
  meetLink: string
  hasGoogle: boolean
  googleSubject: string
  googleCalendarID: string
  hasChat: boolean
  chatId?: number
  chatTitle?: string
}

export type IntegrationsView = {
  hasGoogle: boolean
  googleSubject: string
  googleCalendarID: string
  meetLink: string
  tz: string
}

export type GoogleVerifyResult = {
  ok: boolean
  calendarSummary?: string
  timeZone?: string
  accessRole?: string
}

export type GoogleVerifyError =
  | "google_sa_invalid"
  | "google_subject_invalid"
  | "google_calendar_not_accessible"
  | "google_api_disabled"
  | "google_not_configured"

export type ChatStatus = { linked: boolean; chatId?: number; chatTitle?: string }

export type Member = {
  id: string
  fullName: string
  telegramUsername: string
  role: string
}

export type AuditEntry = {
  id: string
  actorEmail: string
  actorTelegramId: number
  action: string
  targetKind: string
  targetId: string
  details: Record<string, unknown>
  createdAt: string
}
