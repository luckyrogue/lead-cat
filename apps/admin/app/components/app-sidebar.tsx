import {
  LayoutDashboard,
  Mailbox,
  Paw,
  Settings,
  Users,
  Video,
} from "@leadcat/ui"
import { Link, useLocation } from "react-router"

import { SidebarUserCard } from "~/components/sidebar-user-card"
import { cn } from "~/shared/lib/cn"
import type { Me } from "~/shared/auth/types"

const navItems = [
  { href: "/", label: "Dashboard", icon: LayoutDashboard },
  { href: "/members", label: "Members", icon: Users },
  { href: "/invites", label: "Invites", icon: Mailbox },
  { href: "/meetings", label: "Meetings", icon: Video },
  { href: "/settings", label: "Settings", icon: Settings },
]

type AppSidebarProps = {
  me: Me
}

export function AppSidebar({ me }: AppSidebarProps) {
  const location = useLocation()

  return (
    <aside className="surface-card flex flex-col overflow-hidden rounded-[calc(var(--radius)*1.6)] xl:h-full xl:min-h-0">
      <div className="shrink-0 border-b border-border/70 px-5 py-5">
        <Link
          to="/"
          className="flex items-center gap-2 font-bold text-kitty-800"
        >
          <span className="grid size-9 place-items-center rounded-2xl bg-coral-400 text-white">
            <Paw className="size-5" />
          </span>
          Lead Cat
        </Link>
      </div>

      <nav className="flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto p-3">
        {navItems.map((item) => {
          const isActive =
            item.href === "/"
              ? location.pathname === "/"
              : location.pathname.startsWith(item.href)
          const Icon = item.icon

          return (
            <Link
              key={item.href}
              to={item.href}
              className={cn(
                "flex items-center gap-3 rounded-[calc(var(--radius)*0.95)] border border-transparent px-3 py-2.5 transition-all duration-200",
                isActive
                  ? "border-border/80 bg-sidebar-accent text-sidebar-accent-foreground"
                  : "text-muted-foreground hover:border-border/70 hover:bg-background/70 hover:text-foreground"
              )}
            >
              <span
                className={cn(
                  "rounded-[calc(var(--radius)*0.7)] border border-border/70 bg-background/80 p-2",
                  isActive && "border-primary/15 bg-primary/10 text-primary"
                )}
              >
                <Icon className="size-4" />
              </span>
              <span className="text-sm font-medium">{item.label}</span>
            </Link>
          )
        })}
      </nav>

      <SidebarUserCard me={me} />
    </aside>
  )
}
