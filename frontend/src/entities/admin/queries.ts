import { useQuery } from "@tanstack/react-query"
import {
  getAuditLog, getChatStatus, getIntegrations, getMembers,
  getWorkspaceStatus,
  type AuditQuery,
} from "./api"

export const adminKeys = {
  all: ["admin"] as const,
  workspace: () => ["admin", "workspace"] as const,
  integrations: () => ["admin", "integrations"] as const,
  chat: () => ["admin", "chat"] as const,
  members: () => ["admin", "members"] as const,
  audit: (q: AuditQuery) => ["admin", "audit", q] as const,
}

export function useWorkspaceStatus() {
  return useQuery({ queryKey: adminKeys.workspace(), queryFn: getWorkspaceStatus })
}
export function useIntegrations() {
  return useQuery({ queryKey: adminKeys.integrations(), queryFn: getIntegrations })
}
export function useChatStatus() {
  return useQuery({ queryKey: adminKeys.chat(), queryFn: getChatStatus })
}
export function useMembers() {
  return useQuery({ queryKey: adminKeys.members(), queryFn: getMembers })
}
export function useAuditLog(q: AuditQuery = {}) {
  return useQuery({ queryKey: adminKeys.audit(q), queryFn: () => getAuditLog(q) })
}
