import { useQuery } from "@tanstack/react-query"
import {
  getAuditLog, getChatStatus, getIntegrations, getMembers,
  getScenarioRuns, getScenarios, getWorkspaceStatus,
  type AuditQuery,
} from "./api"

export const adminKeys = {
  all: ["admin"] as const,
  workspace: () => ["admin", "workspace"] as const,
  integrations: () => ["admin", "integrations"] as const,
  chat: () => ["admin", "chat"] as const,
  members: () => ["admin", "members"] as const,
  scenarios: () => ["admin", "scenarios"] as const,
  scenarioRuns: (id: string) => ["admin", "scenarios", id, "runs"] as const,
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
export function useScenarios() {
  return useQuery({ queryKey: adminKeys.scenarios(), queryFn: getScenarios })
}
export function useScenarioRuns(id: string) {
  return useQuery({ queryKey: adminKeys.scenarioRuns(id), queryFn: () => getScenarioRuns(id) })
}
export function useAuditLog(q: AuditQuery = {}) {
  return useQuery({ queryKey: adminKeys.audit(q), queryFn: () => getAuditLog(q) })
}
