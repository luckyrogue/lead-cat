import { apiFetch } from "~/shared/api/client"
import type { MeetingScope } from "~/shared/api/query-keys"
import type {
  Conflict,
  CreateMeetingInput,
  FreeSlot,
  Meeting,
  MeetingMutationScope,
  OccurrenceConflicts,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export async function fetchMyMeetings(scope: MeetingScope): Promise<Meeting[]> {
  const res = await apiFetch<{ meetings: Meeting[] }>("/api/miniapp/meetings", {
    params: { scope },
  })
  return res.meetings ?? []
}

export async function fetchSchedule(email: string, scope: MeetingScope): Promise<Meeting[]> {
  const res = await apiFetch<{ meetings: Meeting[] }>("/api/miniapp/schedule", {
    params: { email, scope },
  })
  return res.meetings ?? []
}

export async function createMeeting(input: CreateMeetingInput): Promise<Meeting> {
  const res = await apiFetch<{ meeting: Meeting }>("/api/miniapp/meetings", {
    method: "POST",
    body: input,
  })
  return res.meeting
}

export async function updateMeeting(
  id: string,
  input: UpdateMeetingInput,
  scope: MeetingMutationScope = "this"
): Promise<Meeting> {
  const res = await apiFetch<{ meeting: Meeting }>(`/api/miniapp/meetings/${id}`, {
    method: "PATCH",
    params: { scope },
    body: input,
  })
  return res.meeting
}

export async function deleteMeeting(
  id: string,
  scope: MeetingMutationScope = "this"
): Promise<void> {
  await apiFetch<void>(`/api/miniapp/meetings/${id}`, {
    method: "DELETE",
    params: { scope },
  })
}

export async function addParticipant(id: string, email: string): Promise<Meeting> {
  const res = await apiFetch<{ meeting: Meeting }>(
    `/api/miniapp/meetings/${id}/participants`,
    { method: "POST", body: { email } }
  )
  return res.meeting
}

export async function removeParticipant(id: string, email: string): Promise<Meeting> {
  const res = await apiFetch<{ meeting: Meeting }>(
    `/api/miniapp/meetings/${id}/participants`,
    { method: "DELETE", params: { email } }
  )
  return res.meeting
}

type ConflictsInput = {
  participants: string[]
  date: string
  start: string
  end: string
  exclude_id?: string
}

export async function fetchConflicts(input: ConflictsInput): Promise<OccurrenceConflicts[]> {
  const res = await apiFetch<{ occurrences: OccurrenceConflicts[] }>("/api/miniapp/conflicts", {
    method: "POST",
    body: input,
  })
  return res.occurrences ?? []
}

export function flattenConflicts(occurrences: OccurrenceConflicts[]): Conflict[] {
  return occurrences.flatMap((o) => o.conflicts)
}

type FreeSlotsInput = {
  participants: string[]
  from: string
  to: string
  duration_mins: number
}

export async function fetchFreeSlots(input: FreeSlotsInput): Promise<FreeSlot[]> {
  const res = await apiFetch<{ slots: FreeSlot[] }>("/api/miniapp/free-slots", {
    method: "POST",
    body: input,
  })
  return res.slots ?? []
}
