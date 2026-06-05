import type { TmaUserRole } from "@/features/auth/session"

export type RoleRequirement =
  | TmaUserRole
  | {
      roles: readonly TmaUserRole[]
      mode?: "any" | "all"
      deniedCode?: string
    }

export function matchesRoleRequirement(
  userRole: TmaUserRole | null | undefined,
  requirement: RoleRequirement | null | undefined
): boolean {
  if (!requirement) {
    return true
  }
  if (!userRole) {
    return false
  }

  if (typeof requirement === "string") {
    return userRole === requirement
  }

  if (requirement.roles.length === 0) {
    return true
  }

  if (requirement.mode === "all") {
    return requirement.roles.every((role) => userRole === role)
  }

  return requirement.roles.some((role) => userRole === role)
}

export function getRoleDeniedCode(
  requirement: RoleRequirement | null | undefined
): string | null {
  if (!requirement) {
    return null
  }

  if (typeof requirement === "string") {
    return requirement
  }

  return requirement.deniedCode ?? requirement.roles[0] ?? null
}
