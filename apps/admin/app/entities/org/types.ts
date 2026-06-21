import type {
  Org,
  OrgInvite as ApiOrgInvite,
  OrgMember as ApiOrgMember,
} from "@leadcat/api-client"

export type OrgRole = ApiOrgMember["role"]

export type Organization = Org

export type OrgMemberStatus = ApiOrgMember["status"]

export type OrgMember = ApiOrgMember

export type OrgInvite = ApiOrgInvite
