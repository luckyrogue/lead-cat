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
