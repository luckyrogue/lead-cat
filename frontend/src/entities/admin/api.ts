import { apiFetch } from "@/shared/api/client"
import type {
  AuditEntry,
  ChatStatus,
  IntegrationsView,
  Member,
  Scenario,
  ScenarioRun,
  WorkspaceStatus,
} from "./types"

type AdminWorkspaceDTO = {
  id: string; name: string; tz: string; meet_link: string
  has_google: boolean; google_subject: string; google_calendar_id: string
  has_chat: boolean; chat_id?: number; chat_title?: string
}
type AdminIntegrationsDTO = {
  has_google: boolean; google_subject: string; google_calendar_id: string
  meet_link: string; tz: string
}
type AdminChatStatusDTO = { linked: boolean; chat_id?: number; chat_title?: string }
type AdminMemberDTO = {
  id: string; full_name: string; telegram_username: string; role: string
  github_login?: string; gitlab_login?: string
}
type AdminScenarioDTO = {
  id: string; name: string; enabled: boolean; schedule: string; last_run_at?: string
}
type AdminScenarioRunDTO = {
  id: string; scenario_id: string; status: string; started_at: string
  finished_at?: string; error?: string
}
type AdminAuditDTO = {
  id: string; actor_email: string; actor_telegram_id: number
  action: string; target_kind: string; target_id: string
  details: Record<string, unknown>; created_at: string
}

export async function getWorkspaceStatus(): Promise<WorkspaceStatus> {
  const d = await apiFetch<AdminWorkspaceDTO>("/api/tma/admin/workspace")
  return {
    id: d.id, name: d.name, tz: d.tz, meetLink: d.meet_link,
    hasGoogle: d.has_google, googleSubject: d.google_subject, googleCalendarID: d.google_calendar_id,
    hasChat: d.has_chat, chatId: d.chat_id, chatTitle: d.chat_title,
  }
}

export async function getIntegrations(): Promise<IntegrationsView> {
  const d = await apiFetch<AdminIntegrationsDTO>("/api/tma/admin/integrations")
  return {
    hasGoogle: d.has_google, googleSubject: d.google_subject, googleCalendarID: d.google_calendar_id,
    meetLink: d.meet_link, tz: d.tz,
  }
}

export async function getChatStatus(): Promise<ChatStatus> {
  const d = await apiFetch<AdminChatStatusDTO>("/api/tma/admin/chat/status")
  return { linked: d.linked, chatId: d.chat_id, chatTitle: d.chat_title }
}

export async function getMembers(): Promise<Member[]> {
  const d = await apiFetch<{ members: AdminMemberDTO[] }>("/api/tma/admin/members")
  return d.members.map((m) => ({
    id: m.id, fullName: m.full_name, telegramUsername: m.telegram_username, role: m.role,
    githubLogin: m.github_login, gitlabLogin: m.gitlab_login,
  }))
}

export async function getScenarios(): Promise<Scenario[]> {
  const d = await apiFetch<{ scenarios: AdminScenarioDTO[] }>("/api/tma/admin/scenarios")
  return d.scenarios.map((s) => ({
    id: s.id, name: s.name, enabled: s.enabled, schedule: s.schedule, lastRunAt: s.last_run_at,
  }))
}

export async function getScenarioRuns(scenarioId: string): Promise<ScenarioRun[]> {
  const d = await apiFetch<{ runs: AdminScenarioRunDTO[] }>(`/api/tma/admin/scenarios/${scenarioId}/runs`)
  return d.runs.map((r) => ({
    id: r.id, scenarioId: r.scenario_id, status: r.status, startedAt: r.started_at,
    finishedAt: r.finished_at, error: r.error,
  }))
}

export type AuditQuery = { limit?: number; action?: string; actor?: string }

export async function getAuditLog(q: AuditQuery = {}): Promise<AuditEntry[]> {
  const params = new URLSearchParams()
  if (q.limit) params.set("limit", String(q.limit))
  if (q.action) params.set("action", q.action)
  if (q.actor) params.set("actor", q.actor)
  const qs = params.toString()
  const d = await apiFetch<{ entries: AdminAuditDTO[] }>(
    `/api/tma/admin/audit${qs ? "?" + qs : ""}`
  )
  return d.entries.map((e) => ({
    id: e.id, actorEmail: e.actor_email, actorTelegramId: e.actor_telegram_id,
    action: e.action, targetKind: e.target_kind, targetId: e.target_id,
    details: e.details, createdAt: e.created_at,
  }))
}
