import type { I18nKey } from "@/shared/tma/i18n"
import type { TabKey } from "@/shared/tma/types"
import {
  moduleKeyFromPath,
  tabKeyFromModule,
  type TmaModulePolicy,
  type TmaTabIcon,
} from "@/shared/auth/module-policies"

export function activeTabFromPath(pathname: string): TabKey {
  return tabKeyFromModule(moduleKeyFromPath(pathname))
}

export function tabIconName(
  icon: TmaTabIcon
): "home" | "calendar" | "search" | "user" {
  if (icon === "calendar") return "calendar"
  if (icon === "search") return "search"
  if (icon === "user" || icon === "zap") return "user"
  return "home"
}

export function buildTabBarItems(
  modules: TmaModulePolicy[],
  t: (key: I18nKey) => string
): (
  | {
      key: TabKey
      to: string
      icon: "home" | "calendar" | "search" | "user"
      label: string
    }
  | { key: "_fab" }
)[] {
  const items: (
    | {
        key: TabKey
        to: string
        icon: "home" | "calendar" | "search" | "user"
        label: string
      }
    | { key: "_fab" }
  )[] = []

  modules.forEach((module, index) => {
    items.push({
      key: tabKeyFromModule(module.key),
      to: module.href,
      icon: tabIconName(module.icon),
      label: t(module.labelKey),
    })
    if (index === 1) {
      items.push({ key: "_fab" })
    }
  })

  return items
}
