import { toMeeting } from "@/entities/meeting/api"
import type { Meeting } from "@/entities/meeting/types"
import { apiFetch } from "@/shared/api/client"

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

export type MeetingInput = {
  dept: string
  type: string
  host: string
  date: string // YYYY-MM-DD
  start: string // HH:MM
  end: string // HH:MM
  recurrence: string
  desc: string
  participants: string[] // emails
  recurrence_until?: string
  recurrence_days?: number[]
}

export type MeetingPatch = Partial<{
  dept: string
  type: string
  host: string
  date: string
  start: string
  end: string
  desc: string
}>

export async function createMeeting(input: MeetingInput): Promise<Meeting> {
  const data = await apiFetch<{ meeting: MeetingDTO }>("/miniapp/meetings", {
    method: "POST",
    body: input,
  })
  return toMeeting(data.meeting)
}

export async function updateMeeting(
  id: string,
  patch: MeetingPatch,
  opts?: { scope?: "this" | "whole" }
): Promise<Meeting> {
  const scope = opts?.scope ?? "this"
  const data = await apiFetch<{ meeting: MeetingDTO }>(`/miniapp/meetings/${id}`, {
    method: "PATCH",
    body: patch,
    params: { scope },
  })
  return toMeeting(data.meeting)
}

export async function deleteMeeting(
  id: string,
  opts?: { scope?: "this" | "whole" }
): Promise<void> {
  const scope = opts?.scope ?? "this"
  await apiFetch<void>(`/miniapp/meetings/${id}`, {
    method: "DELETE",
    params: { scope },
  })
}
