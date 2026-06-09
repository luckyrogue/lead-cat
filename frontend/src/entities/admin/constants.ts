export const ADMIN_AUDIT_PAGE_SIZE = 50

export const AUDIT_ACTIONS = [
  "google_config_updated",
  "google_verified",
  "chat_linked",
  "members_synced",
  "scenario_toggled",
  "scenario_run_started",
] as const

export type AuditAction = (typeof AUDIT_ACTIONS)[number]
