import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createInvite,
  createOrg,
  deleteInvite,
  listInvites,
  listMembers,
  removeMember,
  updateMemberRole,
} from "~/entities/org/api"
import type { OrgRole } from "~/entities/org/types"
import { meQueryKey } from "~/shared/auth/use-me"

export const orgKeys = {
  members: (orgId: string) => ["orgs", orgId, "members"] as const,
  invites: (orgId: string) => ["orgs", orgId, "invites"] as const,
}

export function useMembers(orgId: string | null) {
  return useQuery({
    queryKey: orgKeys.members(orgId ?? ""),
    queryFn: () => listMembers(orgId as string),
    enabled: Boolean(orgId),
  })
}

export function useInvites(orgId: string | null) {
  return useQuery({
    queryKey: orgKeys.invites(orgId ?? ""),
    queryFn: () => listInvites(orgId as string),
    enabled: Boolean(orgId),
  })
}

export function useCreateOrg() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => createOrg(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: meQueryKey })
    },
  })
}

export function useUpdateMemberRole(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { userId: string; role: OrgRole }) =>
      updateMemberRole(orgId, input.userId, input.role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.members(orgId) })
    },
  })
}

export function useRemoveMember(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => removeMember(orgId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.members(orgId) })
    },
  })
}

export function useCreateInvite(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { email: string; role: OrgRole }) =>
      createInvite(orgId, input.email, input.role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.invites(orgId) })
    },
  })
}

export function useDeleteInvite(orgId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (inviteId: string) => deleteInvite(orgId, inviteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orgKeys.invites(orgId) })
    },
  })
}
