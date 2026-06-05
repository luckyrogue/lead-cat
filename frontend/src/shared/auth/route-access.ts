import type { TmaUser } from "@/features/auth/session"
import { ApiError } from "@/shared/api/types"
import {
  getTmaModuleAccessRequirement,
  getTmaModuleDeniedCode,
  type TmaModuleKey,
} from "@/shared/auth/module-policies"
import {
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export function isForbiddenApiError(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 403
}

export function canAccessTmaRoute(
  user: TmaUser | null | undefined,
  moduleKey: TmaModuleKey
): boolean {
  return matchesRoleRequirement(
    user?.role,
    getTmaModuleAccessRequirement(moduleKey)
  )
}

export function isTmaRouteForbidden(
  user: TmaUser | null | undefined,
  moduleKey: TmaModuleKey,
  error?: unknown
): boolean {
  return !canAccessTmaRoute(user, moduleKey) || isForbiddenApiError(error)
}

export function getTmaRouteDeniedCode(moduleKey: TmaModuleKey): string {
  return getTmaModuleDeniedCode(moduleKey) ?? "forbidden"
}

export async function ensureRoleQueryData(
  user: TmaUser | null | undefined,
  requirement: RoleRequirement | null | undefined,
  load: () => Promise<unknown>
) {
  if (!matchesRoleRequirement(user?.role, requirement)) {
    return
  }

  try {
    await load()
  } catch (error) {
    if (!isForbiddenApiError(error)) {
      throw error
    }
  }
}

export async function ensureTmaQueryData(
  user: TmaUser | null | undefined,
  moduleKey: TmaModuleKey,
  load: () => Promise<unknown>
) {
  await ensureRoleQueryData(
    user,
    getTmaModuleAccessRequirement(moduleKey),
    load
  )
}
