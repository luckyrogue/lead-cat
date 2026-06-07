import type { TmaUser, TmaUserRole } from "@/shared/auth/types"

import {
  getRoleDeniedCode,
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export function hasRole(
  user: TmaUser | null | undefined,
  requirement: RoleRequirement | null | undefined
): boolean {
  return matchesRoleRequirement(user?.role, requirement)
}

export function isAdmin(user: TmaUser | null | undefined): boolean {
  return user?.role === "admin"
}

export function hasAnyRole(
  user: TmaUser | null | undefined,
  roles: readonly TmaUserRole[]
): boolean {
  return hasRole(user, { roles, mode: "any" })
}

export function getDeniedRoleCode(
  requirement: RoleRequirement | null | undefined
): string | null {
  return getRoleDeniedCode(requirement)
}
