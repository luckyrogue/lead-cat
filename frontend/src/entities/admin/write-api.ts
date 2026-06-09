import { apiFetch } from "@/shared/api/client"
import type { GoogleVerifyResult } from "./types"

export type IntegrationsPatch = {
  googleSAJson?: string
  googleSubject?: string
  googleCalendarID?: string
  meetLink?: string
  tz?: string
}

export async function createWorkspace(): Promise<{ id: string }> {
  return apiFetch<{ id: string }>("/api/tma/admin/workspace", { method: "POST" })
}

export async function patchIntegrations(p: IntegrationsPatch): Promise<void> {
  await apiFetch("/api/tma/admin/integrations", {
    method: "PATCH",
    body: JSON.stringify({
      google_sa_json: p.googleSAJson,
      google_subject: p.googleSubject,
      google_calendar_id: p.googleCalendarID,
      meet_link: p.meetLink,
      tz: p.tz,
    }),
  })
}

export async function verifyIntegrations(): Promise<GoogleVerifyResult> {
  const d = await apiFetch<{
    ok: boolean
    calendar_summary?: string
    time_zone?: string
    access_role?: string
  }>("/api/tma/admin/integrations/verify", { method: "POST" })
  return {
    ok: d.ok, calendarSummary: d.calendar_summary,
    timeZone: d.time_zone, accessRole: d.access_role,
  }
}

export async function linkChat(chatId: number, chatTitle?: string): Promise<void> {
  await apiFetch("/api/tma/admin/chat/link", {
    method: "POST",
    body: JSON.stringify({ chat_id: chatId, chat_title: chatTitle }),
  })
}

export async function syncChatMembers(): Promise<{ added: number }> {
  return apiFetch<{ added: number }>("/api/tma/admin/members/sync-chat", { method: "POST" })
}

export async function patchScenario(id: string, enabled: boolean): Promise<void> {
  await apiFetch(`/api/tma/admin/scenarios/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  })
}

export async function runScenario(id: string): Promise<{ runId: string }> {
  const d = await apiFetch<{ run_id: string }>(
    `/api/tma/admin/scenarios/${id}/run`, { method: "POST" }
  )
  return { runId: d.run_id }
}
