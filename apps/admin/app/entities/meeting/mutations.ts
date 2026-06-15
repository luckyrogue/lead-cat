import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  addParticipant,
  changeSeriesEnd,
  createMeeting,
  deleteMeeting,
  removeParticipant,
  updateMeeting,
} from "~/entities/meeting/api"
import { meetingKeys } from "~/entities/meeting/queries"
import type {
  CreateMeetingInput,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

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

export function useChangeSeriesEnd(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { meetingId: string; until: string }) =>
      changeSeriesEnd(orgId, input.meetingId, input.until),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meetingKeys.list(orgId) })
    },
  })
}
