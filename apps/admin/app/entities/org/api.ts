import { api } from "~/shared/api/client"
import type {
  Organization,
  OrgInvite,
  OrgMember,
  OrgRole,
} from "~/entities/org/types"

export async function createOrg(name: string): Promise<Organization> {
  const { data } = await api.post<Organization>("/api/orgs", { name })
  return data
}

export async function listOrgs(): Promise<Organization[]> {
  const { data } = await api.get<{ organizations: Organization[] }>("/api/orgs")
  return data.organizations ?? []
}

export async function listMembers(orgId: string): Promise<OrgMember[]> {
  const { data } = await api.get<{ members: OrgMember[] }>(
    `/api/orgs/${orgId}/members`
  )
  return data.members ?? []
}

export async function updateMemberRole(
  orgId: string,
  userId: string,
  role: OrgRole
): Promise<void> {
  await api.patch(`/api/orgs/${orgId}/members/${userId}/role`, { role })
}

export async function removeMember(
  orgId: string,
  userId: string
): Promise<void> {
  await api.delete(`/api/orgs/${orgId}/members/${userId}`)
}

export async function listInvites(orgId: string): Promise<OrgInvite[]> {
  const { data } = await api.get<{ invites: OrgInvite[] }>(
    `/api/orgs/${orgId}/invites`
  )
  return data.invites ?? []
}

export async function createInvite(
  orgId: string,
  email: string,
  role: OrgRole
): Promise<OrgInvite> {
  const { data } = await api.post<OrgInvite>(`/api/orgs/${orgId}/invites`, {
    email,
    role,
  })
  return data
}

export async function deleteInvite(
  orgId: string,
  inviteId: string
): Promise<void> {
  await api.delete(`/api/orgs/${orgId}/invites/${inviteId}`)
}
