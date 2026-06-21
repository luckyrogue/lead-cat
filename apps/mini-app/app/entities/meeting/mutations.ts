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

function invalidateMeetingLists(qc: ReturnType<typeof useQueryClient>) {
  void qc.invalidateQueries({ queryKey: meetingKeys.lists() })
}

function invalidateMeetingDetail(
  qc: ReturnType<typeof useQueryClient>,
  id: string
) {
  void qc.invalidateQueries({ queryKey: meetingKeys.detail(id), exact: true })
}

export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMeetingInput) => createMeeting(input),
    onSuccess: () => invalidateMeetingLists(qc),
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
    onSuccess: (_data, { id }) => {
      invalidateMeetingDetail(qc, id)
      invalidateMeetingLists(qc)
    },
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, scope }: { id: string; scope?: MeetingMutationScope }) =>
      deleteMeeting(id, scope),
    onSuccess: () => invalidateMeetingLists(qc),
  })
}

export function useChangeSeriesEnd() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, until }: { id: string; until: string }) =>
      changeSeriesEnd(id, until),
    onSuccess: (_data, { id }) => {
      invalidateMeetingDetail(qc, id)
      invalidateMeetingLists(qc)
    },
  })
}

export function useAddParticipant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, email }: { id: string; email: string }) =>
      addParticipant(id, email),
    onSuccess: (_data, { id }) => {
      invalidateMeetingDetail(qc, id)
      invalidateMeetingLists(qc)
    },
  })
}

export function useRemoveParticipant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, email }: { id: string; email: string }) =>
      removeParticipant(id, email),
    onSuccess: (_data, { id }) => {
      invalidateMeetingDetail(qc, id)
      invalidateMeetingLists(qc)
    },
  })
}
