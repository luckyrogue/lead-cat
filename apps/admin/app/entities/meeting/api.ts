import { api } from "~/shared/api/client"
import type {
  CreateMeetingInput,
  Meeting,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export async function listMeetings(orgId: string): Promise<Meeting[]> {
  const { data } = await api.get<{ meetings: Meeting[] }>(
    `/api/orgs/${orgId}/meetings`
  )
  return data.meetings ?? []
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
  input: UpdateMeetingInput
): Promise<Meeting> {
  const { data } = await api.patch<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}`,
    input
  )
  return data.meeting
}

export async function deleteMeeting(
  orgId: string,
  meetingId: string
): Promise<void> {
  await api.delete(`/api/orgs/${orgId}/meetings/${meetingId}`)
}
