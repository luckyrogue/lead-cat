export type MyJoinRequest = { organization_id: string; org_name: string; status: string }
export type JoinResult = { already_member?: boolean; status?: string; organization_id: string }
export type OrgJoinRequest = { request_id: string; user_id: string; name: string; email: string; created_at: string }
