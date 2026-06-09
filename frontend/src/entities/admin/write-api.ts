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
  return apiFetch<{ id: string }>("/miniapp/admin/workspace", { method: "POST" })
}

export async function patchIntegrations(p: IntegrationsPatch): Promise<void> {
  await apiFetch("/miniapp/admin/integrations", {
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
  }>("/miniapp/admin/integrations/verify", { method: "POST" })
  return {
    ok: d.ok, calendarSummary: d.calendar_summary,
    timeZone: d.time_zone, accessRole: d.access_role,
  }
}

export async function linkChat(chatId: number, chatTitle?: string): Promise<void> {
  await apiFetch("/miniapp/admin/chat/link", {
    method: "POST",
    body: JSON.stringify({ chat_id: chatId, chat_title: chatTitle }),
  })
}

export async function syncChatMembers(): Promise<{ added: number }> {
  return apiFetch<{ added: number }>("/miniapp/admin/members/sync-chat", { method: "POST" })
}
