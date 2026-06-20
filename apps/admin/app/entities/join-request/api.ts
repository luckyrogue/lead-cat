import { api } from "~/shared/api/client"
import type { JoinResult, MyJoinRequest, OrgJoinRequest } from "./types"

export async function requestToJoin(slug: string): Promise<JoinResult> {
  const { data } = await api.post<JoinResult>(
    "/api/auth/web/me/join-requests",
    { slug }
  )
  return data
}

export async function listMyJoinRequests(): Promise<MyJoinRequest[]> {
  const { data } = await api.get<MyJoinRequest[]>(
    "/api/auth/web/me/join-requests"
  )
  return data
}

export async function listOrgJoinRequests(
  orgId: string
): Promise<OrgJoinRequest[]> {
  const { data } = await api.get<OrgJoinRequest[]>(
    `/api/orgs/${orgId}/join-requests`
  )
  return data
}

export async function acceptJoinRequest(
  orgId: string,
  rid: string
): Promise<void> {
  await api.post(`/api/orgs/${orgId}/join-requests/${rid}/accept`, {})
}

export async function declineJoinRequest(
  orgId: string,
  rid: string
): Promise<void> {
  await api.post(`/api/orgs/${orgId}/join-requests/${rid}/decline`, {})
}
