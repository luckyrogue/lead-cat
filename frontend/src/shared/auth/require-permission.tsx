import type { ReactNode } from "react"

import type { TmaUser } from "@/features/auth/session"
import { PermissionDenied } from "@/components/auth/permission-denied"
import { hasRole } from "@/shared/auth/permissions"
import type { RoleRequirement } from "@/shared/auth/permission-requirement"

type RequireTmaRoleProps = {
  user: TmaUser | null | undefined
  requirement: RoleRequirement | null | undefined
  fallback?: ReactNode
  children: ReactNode
}

export function RequireTmaRole({
  user,
  requirement,
  fallback,
  children,
}: RequireTmaRoleProps) {
  if (!hasRole(user, requirement)) {
    return fallback ?? <PermissionDenied />
  }
  return children
}
