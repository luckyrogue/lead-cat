import { useMutation, useQueryClient } from "@tanstack/react-query"

import {
  createMeeting,
  deleteMeeting,
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
