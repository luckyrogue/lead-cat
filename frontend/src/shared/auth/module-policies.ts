import type { MiniAppUser } from "@/shared/auth/types"
import type { TabKey } from "@/shared/miniapp/types"
import {
  getRoleDeniedCode,
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export type MiniAppModuleKey =
  | "home"
  | "meetings"
  | "checker"
  | "profile"
  | "admin"

export type MiniAppModuleActionKey = "read" | "manage"

export type MiniAppTabIcon = "home" | "calendar" | "search" | "user" | "zap"

export type MiniAppModulePolicy = {
  key: MiniAppModuleKey
  href: string
  labelKey:
    | "nav_home"
    | "nav_meetings"
    | "nav_checker"
    | "nav_profile"
  icon: MiniAppTabIcon
  showInTabBar?: boolean
  access: RoleRequirement | null
  actions?: Partial<Record<MiniAppModuleActionKey, RoleRequirement>>
}

export const miniappModulePolicies: MiniAppModulePolicy[] = [
  {
    key: "home",
    href: "/",
    labelKey: "nav_home",
    icon: "home",
    showInTabBar: true,
    access: null,
  },
  {
    key: "meetings",
    href: "/meetings",
    labelKey: "nav_meetings",
    icon: "calendar",
    showInTabBar: true,
    access: null,
  },
  {
    key: "checker",
    href: "/checker",
    labelKey: "nav_checker",
    icon: "search",
    showInTabBar: true,
    access: null,
  },
  {
    key: "profile",
    href: "/profile",
    labelKey: "nav_profile",
    icon: "user",
    showInTabBar: true,
    access: null,
  },
  {
    key: "admin",
    href: "/profile/admin",
    labelKey: "nav_profile",
    icon: "user",
    showInTabBar: false,
    access: "admin",
    actions: { read: "admin", manage: "admin" },
  },
]

export function getMiniAppModulePolicy(key: MiniAppModuleKey): MiniAppModulePolicy {
  const policy = miniappModulePolicies.find((item) => item.key === key)
  if (!policy) {
    throw new Error(`unknown tma module: ${key}`)
  }
  return policy
}

export function getMiniAppModuleAccessRequirement(
  key: MiniAppModuleKey
): RoleRequirement | null {
  return getMiniAppModulePolicy(key).access
}

export function getMiniAppModuleDeniedCode(key: MiniAppModuleKey): string | null {
  return getRoleDeniedCode(getMiniAppModuleAccessRequirement(key))
}

export function canAccessMiniAppModule(
  user: MiniAppUser | null | undefined,
  key: MiniAppModuleKey
): boolean {
  return matchesRoleRequirement(user?.role, getMiniAppModuleAccessRequirement(key))
}

export function canAccessMiniAppModuleAction(
  user: MiniAppUser | null | undefined,
  key: MiniAppModuleKey,
  action: MiniAppModuleActionKey
): boolean {
  const policy = getMiniAppModulePolicy(key)
  const requirement = policy.actions?.[action] ?? policy.access
  return matchesRoleRequirement(user?.role, requirement)
}

export function getVisibleTabBarModules(
  user: MiniAppUser | null | undefined
): MiniAppModulePolicy[] {
  return miniappModulePolicies.filter(
    (policy) =>
      policy.showInTabBar !== false && canAccessMiniAppModule(user, policy.key)
  )
}

export function canAccessMiniAppAdmin(user: MiniAppUser | null | undefined): boolean {
  return canAccessMiniAppModule(user, "admin")
}

export function moduleKeyFromPath(pathname: string): MiniAppModuleKey {
  if (pathname.startsWith("/profile/admin")) return "admin"
  if (pathname.startsWith("/profile")) return "profile"
  if (pathname.startsWith("/meetings")) return "meetings"
  if (pathname.startsWith("/checker")) return "checker"
  return "home"
}

export function tabKeyFromModule(key: MiniAppModuleKey): TabKey {
  if (key === "admin") return "profile"
  return key as TabKey
}
