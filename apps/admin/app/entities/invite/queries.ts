import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { acceptInvite, declineInvite, listMyInvites } from "./api"
import { meQueryKey } from "~/shared/auth/use-me"

export const myInvitesKey = ["invites", "mine"] as const

export function useMyInvites() {
  return useQuery({
    queryKey: myInvitesKey,
    queryFn: listMyInvites,
    retry: false,
  })
}

export function useAcceptInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: acceptInvite,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meQueryKey })
      void qc.invalidateQueries({ queryKey: myInvitesKey })
    },
  })
}

export function useDeclineInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: declineInvite,
    onSuccess: () => void qc.invalidateQueries({ queryKey: myInvitesKey }),
  })
}
