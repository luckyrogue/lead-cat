import { Link, useRouterState } from "@tanstack/react-router"
import { useTmaAuth } from "@/shared/auth/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import { getVisibleTabBarModules } from "@/shared/auth/module-policies"
import { cn } from "@/shared/lib/cn"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { activeTabFromPath, buildTabBarItems } from "./tab-bar-helpers"

export function TabBar() {
  const { t } = useTmaApp()
  const { user } = useTmaAuth()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const tab = activeTabFromPath(pathname)
  const items = buildTabBarItems(getVisibleTabBarModules(user), t)

  return (
    <div className="tma-tabbar flex items-start border-t border-tma-border bg-tma-tg-bar">
      {items.map((it) => {
        if (it.key === "_fab") {
          return (
            <div
              key="_fab"
              className="relative flex flex-1 justify-center"
            >
              <Link
                to="/meetings/create"
                className="absolute -top-[30px] flex size-[58px] cursor-pointer items-center justify-center rounded-[20px] border-4 border-tma-tg-bar bg-tma-accent shadow-[0_8px_20px_var(--tma-accent-glow)] transition-transform duration-150 active:scale-90 active:rotate-90"
              >
                <CatIcon
                  name="plus"
                  size={28}
                  className="text-tma-accent-text"
                  sw={2.6}
                />
              </Link>
            </div>
          )
        }
        const active = it.key === tab
        return (
          <Link
            key={it.key}
            to={it.to}
            className={cn(
              "flex flex-1 cursor-pointer flex-col items-center gap-[3px] border-none bg-transparent p-0 no-underline transition-colors duration-[180ms]",
              active ? "text-tma-accent" : "text-tma-muted"
            )}
          >
            <div
              className={cn(
                "relative transition-transform duration-200 ease-[cubic-bezier(.34,1.56,.64,1)]",
                active && "-translate-y-0.5"
              )}
            >
              <CatIcon
                name={it.icon}
                size={24}
                className={active ? "text-tma-accent" : "text-tma-muted"}
                sw={active ? 2.3 : 1.9}
              />
            </div>
            <span
              className={cn(
                "font-display text-[10.5px]",
                active ? "font-extrabold" : "font-semibold"
              )}
            >
              {it.label}
            </span>
          </Link>
        )
      })}
    </div>
  )
}
