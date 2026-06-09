import type { ReactNode } from "react"

import type { MiniAppUser } from "@/shared/auth/types"
import { PermissionDenied } from "@/components/auth/permission-denied"
import { hasRole } from "@/shared/auth/permissions"
import type { RoleRequirement } from "@/shared/auth/permission-requirement"

type RequireMiniAppRoleProps = {
  user: MiniAppUser | null | undefined
  requirement: RoleRequirement | null | undefined
  fallback?: ReactNode
  children: ReactNode
}

export function RequireMiniAppRole({
  user,
  requirement,
  fallback,
  children,
}: RequireMiniAppRoleProps) {
  if (!hasRole(user, requirement)) {
    return fallback ?? <PermissionDenied />
  }
  return children
}
