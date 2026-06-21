import {
  CalendarClock,
  cn,
  Home,
  ListChecks,
  User,
  type LucideIcon,
} from "@leadcat/ui"
import { NavLink } from "react-router"

import { useT } from "~/shared/i18n/context"

type Tab = {
  to: string
  labelKey: string
  icon: LucideIcon
  end?: boolean
}

const TAB_DEFS: Tab[] = [
  { to: "/", labelKey: "nav.home", icon: Home, end: true },
  { to: "/meetings", labelKey: "nav.meetings", icon: CalendarClock },
  { to: "/checker", labelKey: "nav.checker", icon: ListChecks },
  { to: "/profile", labelKey: "nav.profile", icon: User },
]

export function TabBar() {
  const t = useT()

  return (
    <nav
      aria-label={t("nav.mainAriaLabel")}
      className="fixed inset-x-0 bottom-0 z-40 mx-auto w-full max-w-md border-t border-border/60 bg-[var(--tab-bar-bg,var(--background))]/95 px-2 pb-[env(safe-area-inset-bottom)] backdrop-blur"
    >
      <ul className="flex items-stretch justify-around">
        {TAB_DEFS.map((tab) => {
          const Icon = tab.icon
          return (
            <li key={tab.to} className="flex-1">
              <NavLink
                to={tab.to}
                end={tab.end}
                className={({ isActive }) =>
                  cn(
                    "flex flex-col items-center gap-1 py-2.5 text-xs font-medium transition-colors",
                    isActive ? "text-primary" : "text-muted-foreground"
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <span
                      className={cn(
                        "rounded-full px-3 py-1 transition-colors",
                        isActive && "bg-primary/10"
                      )}
                    >
                      <Icon className="size-5" />
                    </span>
                    <span>{t(tab.labelKey)}</span>
                  </>
                )}
              </NavLink>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
