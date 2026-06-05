import type { TmaUser } from "@/features/auth/session"
import type { TabKey } from "@/shared/tma/types"
import {
  getRoleDeniedCode,
  matchesRoleRequirement,
  type RoleRequirement,
} from "@/shared/auth/permission-requirement"

export type TmaModuleKey =
  | "home"
  | "meetings"
  | "checker"
  | "auto"
  | "profile"
  | "admin"

export type TmaModuleActionKey = "read" | "manage"

export type TmaTabIcon = "home" | "calendar" | "search" | "user" | "zap"

export type TmaModulePolicy = {
  key: TmaModuleKey
  href: string
  labelKey:
    | "nav_home"
    | "nav_meetings"
    | "nav_checker"
    | "nav_auto"
    | "nav_profile"
  icon: TmaTabIcon
  showInTabBar?: boolean
  access: RoleRequirement | null
  actions?: Partial<Record<TmaModuleActionKey, RoleRequirement>>
}

export const tmaModulePolicies: TmaModulePolicy[] = [
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
    key: "auto",
    href: "/auto",
    labelKey: "nav_auto",
    icon: "zap",
    showInTabBar: false,
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

/** @deprecated use tmaModulePolicies */
export const tmaTabPolicies = tmaModulePolicies.filter((p) =>
  ["home", "meetings", "checker", "auto", "profile"].includes(p.key)
)

export function getTmaModulePolicy(key: TmaModuleKey): TmaModulePolicy {
  const policy = tmaModulePolicies.find((item) => item.key === key)
  if (!policy) {
    throw new Error(`unknown tma module: ${key}`)
  }
  return policy
}

export function getTmaModuleAccessRequirement(
  key: TmaModuleKey
): RoleRequirement | null {
  return getTmaModulePolicy(key).access
}

export function getTmaModuleDeniedCode(key: TmaModuleKey): string | null {
  return getRoleDeniedCode(getTmaModuleAccessRequirement(key))
}

export function canAccessTmaModule(
  user: TmaUser | null | undefined,
  key: TmaModuleKey
): boolean {
  return matchesRoleRequirement(user?.role, getTmaModuleAccessRequirement(key))
}

export function canAccessTmaModuleAction(
  user: TmaUser | null | undefined,
  key: TmaModuleKey,
  action: TmaModuleActionKey
): boolean {
  const policy = getTmaModulePolicy(key)
  const requirement = policy.actions?.[action] ?? policy.access
  return matchesRoleRequirement(user?.role, requirement)
}

export function getVisibleTabBarModules(
  user: TmaUser | null | undefined
): TmaModulePolicy[] {
  return tmaModulePolicies.filter(
    (policy) =>
      policy.showInTabBar !== false && canAccessTmaModule(user, policy.key)
  )
}

/** @deprecated use getVisibleTabBarModules */
export function visibleTmaTabs(user: TmaUser | null | undefined): TmaModulePolicy[] {
  return tmaModulePolicies.filter(
    (policy) =>
      ["home", "meetings", "checker", "auto", "profile"].includes(policy.key) &&
      canAccessTmaModule(user, policy.key)
  )
}

export function canAccessTmaAdmin(user: TmaUser | null | undefined): boolean {
  return canAccessTmaModule(user, "admin")
}

export function moduleKeyFromPath(pathname: string): TmaModuleKey {
  if (pathname.startsWith("/profile/admin")) return "admin"
  if (pathname.startsWith("/profile")) return "profile"
  if (pathname.startsWith("/meetings")) return "meetings"
  if (pathname.startsWith("/checker")) return "checker"
  if (pathname.startsWith("/auto")) return "auto"
  return "home"
}

export function tabKeyFromModule(key: TmaModuleKey): TabKey {
  if (key === "admin") return "profile"
  return key as TabKey
}
