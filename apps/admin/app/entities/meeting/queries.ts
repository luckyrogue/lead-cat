import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  addParticipant,
  createMeeting,
  deleteMeeting,
  getMeeting,
  listMeetings,
  removeParticipant,
  updateMeeting,
} from "~/entities/meeting/api"
import type {
  CreateMeetingInput,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export const meetingKeys = {
  list: (orgId: string) => ["orgs", orgId, "meetings"] as const,
  detail: (orgId: string, meetingId: string) =>
    ["orgs", orgId, "meetings", meetingId] as const,
}

export function useMeetings(orgId: string | null) {
  return useQuery({
    queryKey: meetingKeys.list(orgId ?? ""),
    queryFn: () => listMeetings(orgId as string),
    enabled: Boolean(orgId),
  })
}

export function useMeeting(orgId: string, meetingId: string | null) {
  return useQuery({
    queryKey: meetingKeys.detail(orgId, meetingId ?? ""),
    queryFn: () => getMeeting(orgId, meetingId as string),
    enabled: Boolean(orgId) && Boolean(meetingId),
  })
}

function useParticipantMutation(
  orgId: string,
  mutationFn: (input: { meetingId: string; email: string }) => Promise<unknown>
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: meetingKeys.detail(orgId, variables.meetingId),
      })
      queryClient.invalidateQueries({ queryKey: meetingKeys.list(orgId) })
    },
  })
}

export function useAddParticipant(orgId: string) {
  return useParticipantMutation(orgId, ({ meetingId, email }) =>
    addParticipant(orgId, meetingId, email)
  )
}

export function useRemoveParticipant(orgId: string) {
  return useParticipantMutation(orgId, ({ meetingId, email }) =>
    removeParticipant(orgId, meetingId, email)
  )
}

export function useCreateMeeting(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMeetingInput) => createMeeting(orgId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meetingKeys.list(orgId) })
    },
  })
}

export function useUpdateMeeting(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: {
      meetingId: string
      values: UpdateMeetingInput
      scope: MeetingScope
    }) => updateMeeting(orgId, input.meetingId, input.values, input.scope),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meetingKeys.list(orgId) })
    },
  })
}

export function useDeleteMeeting(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { meetingId: string; scope: MeetingScope }) =>
      deleteMeeting(orgId, input.meetingId, input.scope),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meetingKeys.list(orgId) })
    },
  })
}
