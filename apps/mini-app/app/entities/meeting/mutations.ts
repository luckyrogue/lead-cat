import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  addParticipant,
  changeSeriesEnd,
  createMeeting,
  deleteMeeting,
  removeParticipant,
  updateMeeting,
} from "~/entities/meeting/api"
import type {
  CreateMeetingInput,
  MeetingMutationScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"
import { meetingKeys } from "~/shared/api/query-keys"

export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMeetingInput) => createMeeting(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      input,
      scope,
    }: {
      id: string
      input: UpdateMeetingInput
      scope?: MeetingMutationScope
    }) => updateMeeting(id, input, scope),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, scope }: { id: string; scope?: MeetingMutationScope }) =>
      deleteMeeting(id, scope),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}

export function useChangeSeriesEnd() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, until }: { id: string; until: string }) =>
      changeSeriesEnd(id, until),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}

export function useAddParticipant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, email }: { id: string; email: string }) =>
      addParticipant(id, email),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}

export function useRemoveParticipant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, email }: { id: string; email: string }) =>
      removeParticipant(id, email),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meetingKeys.all })
    },
  })
}
