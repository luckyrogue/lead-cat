export type OrgRole = "owner" | "admin" | "member"

export type Organization = {
  id: string
  name: string
  slug: string
}

export type OrgMemberStatus = "active" | "invited"

export type OrgMember = {
  user_id: string | null
  role: OrgRole
  email: string
  name: string
  status: OrgMemberStatus
  invited_email: string | null
  telegram_username: string
}

export type OrgInvite = {
  id: string
  email: string
  role: OrgRole
  expires_at?: string
}
