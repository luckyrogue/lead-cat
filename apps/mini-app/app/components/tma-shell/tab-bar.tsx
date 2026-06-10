import {
  CalendarClock,
  cn,
  Home,
  ListChecks,
  User,
  type LucideIcon,
} from "@leadcat/ui"
import { NavLink } from "react-router"

type Tab = {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
}

const TABS: Tab[] = [
  { to: "/", label: "Home", icon: Home, end: true },
  { to: "/meetings", label: "Meetings", icon: CalendarClock },
  { to: "/checker", label: "Checker", icon: ListChecks },
  { to: "/profile", label: "Profile", icon: User },
]

export function TabBar() {
  return (
    <nav className="fixed inset-x-0 bottom-0 z-40 mx-auto w-full max-w-md border-t border-border/60 bg-background/95 px-2 pb-[env(safe-area-inset-bottom)] backdrop-blur">
      <ul className="flex items-stretch justify-around">
        {TABS.map((tab) => {
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
                    <span>{tab.label}</span>
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
