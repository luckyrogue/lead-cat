import { api } from "~/shared/api/client"
import type { MyInvite } from "./types"

export async function listMyInvites(): Promise<MyInvite[]> {
  const { data } = await api.get<MyInvite[]>("/api/auth/web/me/invites")
  return data
}

export async function acceptInvite(iid: string): Promise<void> {
  await api.post(`/api/auth/web/me/invites/${iid}/accept`, {})
}

export async function declineInvite(iid: string): Promise<void> {
  await api.post(`/api/auth/web/me/invites/${iid}/decline`, {})
}
