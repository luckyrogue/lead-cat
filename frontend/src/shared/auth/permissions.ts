import type { MiniAppUser, MiniAppUserRole } from "@/shared/auth/types"

import {
  getRoleDeniedCode,
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export function hasRole(
  user: MiniAppUser | null | undefined,
  requirement: RoleRequirement | null | undefined
): boolean {
  return matchesRoleRequirement(user?.role, requirement)
}

export function isAdmin(user: MiniAppUser | null | undefined): boolean {
  return user?.role === "admin"
}

export function hasAnyRole(
  user: MiniAppUser | null | undefined,
  roles: readonly MiniAppUserRole[]
): boolean {
  return hasRole(user, { roles, mode: "any" })
}

export function getDeniedRoleCode(
  requirement: RoleRequirement | null | undefined
): string | null {
  return getRoleDeniedCode(requirement)
}
