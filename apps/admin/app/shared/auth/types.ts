export type WebUser = {
  id: string
  email: string
  avatar_url: string
  auth_method: string
  timezone?: string
  language?: string
}

export type MeOrganization = {
  id: string
  name: string
  slug: string
}

export type Me = {
  user: WebUser
  organizations: MeOrganization[]
}
