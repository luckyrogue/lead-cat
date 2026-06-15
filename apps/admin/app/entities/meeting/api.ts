import { api } from "~/shared/api/client"
import type {
  CreateMeetingInput,
  Meeting,
  MeetingListFilter,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export async function listMeetings(
  orgId: string,
  filter: MeetingListFilter = {}
): Promise<Meeting[]> {
  const params: Record<string, string> = {}
  if (filter.status && filter.status !== "all") {
    params.status = filter.status
  }
  if (filter.from) {
    params.from = filter.from
  }
  if (filter.to) {
    params.to = filter.to
  }
  if (filter.dept) {
    params.dept = filter.dept
  }
  if (filter.organizer) {
    params.organizer = filter.organizer
  }
  const { data } = await api.get<{ meetings: Meeting[] }>(
    `/api/orgs/${orgId}/meetings`,
    { params }
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

export async function changeSeriesEnd(
  orgId: string,
  meetingId: string,
  until: string
): Promise<Meeting> {
  const { data } = await api.patch<{ meeting: Meeting }>(
    `/api/orgs/${orgId}/meetings/${meetingId}/series-end`,
    { until }
  )
  return data.meeting
}
