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
import { miniappKeys } from "@/shared/api/query-keys"

export function useCreateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: MeetingInput) => createMeeting(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: miniappKeys.all }),
  })
}

export function useUpdateMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      patch,
      scope,
    }: {
      id: string
      patch: MeetingPatch
      scope?: "this" | "whole"
    }) => updateMeeting(id, patch, { scope }),
    onSuccess: () => qc.invalidateQueries({ queryKey: miniappKeys.all }),
  })
}

export function useDeleteMeeting() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: string | { id: string; scope?: "this" | "whole" }) => {
      if (typeof args === "string") return deleteMeeting(args)
      return deleteMeeting(args.id, { scope: args.scope })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: miniappKeys.all }),
  })
}

export function useConflicts() {
  return useMutation({
    mutationFn: (params: ConflictsParams) => fetchConflicts(params),
  })
}
