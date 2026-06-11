import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createMeeting,
  deleteMeeting,
  listMeetings,
  updateMeeting,
} from "~/entities/meeting/api"
import type {
  CreateMeetingInput,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export const meetingKeys = {
  list: (orgId: string) => ["orgs", orgId, "meetings"] as const,
}

export function useMeetings(orgId: string | null) {
  return useQuery({
    queryKey: meetingKeys.list(orgId ?? ""),
    queryFn: () => listMeetings(orgId as string),
    enabled: Boolean(orgId),
  })
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
