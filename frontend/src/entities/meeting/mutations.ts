import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  fetchConflicts,
  type ConflictsParams,
} from "@/entities/meeting/scheduling-api"
import {
  createMeeting,
  deleteMeeting,
  type MeetingInput,
  type MeetingPatch,
  updateMeeting,
} from "@/entities/meeting/write-api"
import { tmaKeys } from "@/shared/api/query-keys"

export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MeetingInput) => createMeeting(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: MeetingPatch }) =>
      updateMeeting(id, patch),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteMeeting(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: tmaKeys.all }),
  })
}

export function useConflicts() {
  return useMutation({
    mutationFn: (params: ConflictsParams) => fetchConflicts(params),
  })
}
