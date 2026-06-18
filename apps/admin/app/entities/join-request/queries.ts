import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  acceptJoinRequest,
  declineJoinRequest,
  listMyJoinRequests,
  listOrgJoinRequests,
  requestToJoin,
} from "./api"
import { orgKeys } from "~/entities/org/queries"

export const myJoinRequestsKey = ["join-requests", "mine"] as const

export const orgJoinRequestsKey = (orgId: string) =>
  ["join-requests", "org", orgId] as const

export function useMyJoinRequests() {
  return useQuery({
    queryKey: myJoinRequestsKey,
    queryFn: listMyJoinRequests,
    retry: false,
  })
}

export function useRequestToJoin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (slug: string) => requestToJoin(slug),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: myJoinRequestsKey })
    },
  })
}

export function useOrgJoinRequests(orgId: string | null) {
  return useQuery({
    queryKey: orgJoinRequestsKey(orgId ?? ""),
    queryFn: () => listOrgJoinRequests(orgId as string),
    enabled: !!orgId,
  })
}

export function useAcceptJoinRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orgId, rid }: { orgId: string; rid: string }) =>
      acceptJoinRequest(orgId, rid),
    onSuccess: (_data, { orgId }) => {
      void qc.invalidateQueries({ queryKey: orgJoinRequestsKey(orgId) })
      void qc.invalidateQueries({ queryKey: orgKeys.members(orgId) })
    },
  })
}

export function useDeclineJoinRequest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orgId, rid }: { orgId: string; rid: string }) =>
      declineJoinRequest(orgId, rid),
    onSuccess: (_data, { orgId }) => {
      void qc.invalidateQueries({ queryKey: orgJoinRequestsKey(orgId) })
    },
  })
}
