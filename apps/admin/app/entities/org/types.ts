export type OrgRole = "owner" | "admin" | "member"

export type Organization = {
  id: string
  name: string
  slug: string
}

export type OrgMember = {
  user_id: string
  role: OrgRole
  invited_email: string
  telegram_username: string
}

export type OrgInvite = {
  id: string
  email: string
  role: OrgRole
  expires_at?: string
}
