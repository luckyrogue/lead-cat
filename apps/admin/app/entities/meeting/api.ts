import { api } from "~/shared/api/client"
import type {
  CreateMeetingInput,
  Meeting,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export async function listMeetings(orgId: string): Promise<Meeting[]> {
  const { data } = await api.get<{ meetings: Meeting[] }>(
    `/api/orgs/${orgId}/meetings`
  )
  return data.meetings ?? []
}

export async function getMeeting(
  orgId: string,
  meetingId: string
): Promise<Meeting> {
  const { data } = await api.get<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}`
  )
  return data.meeting
}

export async function addParticipant(
  orgId: string,
  meetingId: string,
  email: string
): Promise<Meeting> {
  const { data } = await api.post<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}/participants`,
    { email }
  )
  return data.meeting
}

export async function removeParticipant(
  orgId: string,
  meetingId: string,
  email: string
): Promise<Meeting> {
  const { data } = await api.delete<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}/participants`,
    { params: { email } }
  )
  return data.meeting
}

export async function createMeeting(
  orgId: string,
  input: CreateMeetingInput
): Promise<Meeting> {
  const { data } = await api.post<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings`,
    input
  )
  return data.meeting
}

export async function updateMeeting(
  orgId: string,
  meetingId: string,
  input: UpdateMeetingInput,
  scope: MeetingScope = "this"
): Promise<Meeting> {
  const { data } = await api.patch<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}`,
    input,
    { params: { scope } }
  )
  return data.meeting
}

export async function deleteMeeting(
  orgId: string,
  meetingId: string,
  scope: MeetingScope = "this"
): Promise<void> {
  await api.delete(`/api/orgs/${orgId}/meetings/${meetingId}`, {
    params: { scope },
  })
}
