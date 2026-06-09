import type { MiniAppUser } from "@/shared/auth/types"
import { ApiError } from "@/shared/api/types"
import {
  getMiniAppModuleAccessRequirement,
  getMiniAppModuleDeniedCode,
  type MiniAppModuleKey,
} from "@/shared/auth/module-policies"
import {
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export function isForbiddenApiError(error: unknown): error is ApiError {
  return error instanceof ApiError && error.status === 403
}

export function canAccessMiniAppRoute(
  user: MiniAppUser | null | undefined,
  moduleKey: MiniAppModuleKey
): boolean {
  return matchesRoleRequirement(
    user?.role,
    getMiniAppModuleAccessRequirement(moduleKey)
  )
}

export function isMiniAppRouteForbidden(
  user: MiniAppUser | null | undefined,
  moduleKey: MiniAppModuleKey,
  error?: unknown
): boolean {
  return !canAccessMiniAppRoute(user, moduleKey) || isForbiddenApiError(error)
}

export function getMiniAppRouteDeniedCode(moduleKey: MiniAppModuleKey): string {
  return getMiniAppModuleDeniedCode(moduleKey) ?? "forbidden"
}

export async function ensureRoleQueryData(
  user: MiniAppUser | null | undefined,
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

export async function ensureMiniAppQueryData(
  user: MiniAppUser | null | undefined,
  moduleKey: MiniAppModuleKey,
  load: () => Promise<unknown>
) {
  await ensureRoleQueryData(
    user,
    getMiniAppModuleAccessRequirement(moduleKey),
    load
  )
}
