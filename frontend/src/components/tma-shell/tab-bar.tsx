import { Link, useRouterState } from "@tanstack/react-router"
import { useTmaAuth } from "@/features/auth/auth-context"
import { hexToRgba } from "@/shared/tma/palette"
import { useTmaApp } from "@/shared/tma/context"
import { getVisibleTabBarModules } from "@/shared/auth/module-policies"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { activeTabFromPath, buildTabBarItems } from "./tab-bar-helpers"

export function TabBar() {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const tab = activeTabFromPath(pathname)
  const items = buildTabBarItems(getVisibleTabBarModules(user), p.t)

  return (
    <div
      className="tma-tabbar"
      style={{
        background: p.tgBar,
        borderTop: `1px solid ${p.border}`,
        display: "flex",
        alignItems: "flex-start",
      }}
    >
      {items.map((it) => {
        if (it.key === "_fab") {
          return (
            <div
              key="_fab"
              style={{
                flex: 1,
                display: "flex",
                justifyContent: "center",
                position: "relative",
              }}
            >
              <Link
                to="/meetings/create"
                style={{
                  position: "absolute",
                  top: -30,
                  width: 58,
                  height: 58,
                  borderRadius: 20,
                  background: p.accent,
                  border: `4px solid ${p.tgBar}`,
                  cursor: "pointer",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  boxShadow: `0 8px 20px ${hexToRgba(p.accent, 0.45)}`,
                  transition: "transform .15s",
                }}
                onPointerDown={(e) => {
                  e.currentTarget.style.transform = "scale(0.9) rotate(90deg)"
                }}
                onPointerUp={(e) => {
                  e.currentTarget.style.transform = "scale(1)"
                }}
                onPointerLeave={(e) => {
                  e.currentTarget.style.transform = "scale(1)"
                }}
              >
                <CatIcon name="plus" size={28} color={p.accentText} sw={2.6} />
              </Link>
            </div>
          )
        }
        const active = it.key === tab
        return (
          <Link
            key={it.key}
            to={it.to}
            style={{
              flex: 1,
              background: "none",
              border: "none",
              cursor: "pointer",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: 3,
              padding: 0,
              textDecoration: "none",
              color: active ? p.accent : p.muted,
              transition: "color .18s",
            }}
          >
            <div
              style={{
                position: "relative",
                transition: "transform .2s cubic-bezier(.34,1.56,.64,1)",
                transform: active ? "translateY(-2px)" : "none",
              }}
            >
              <CatIcon
                name={it.icon}
                size={24}
                color={active ? p.accent : p.muted}
                sw={active ? 2.3 : 1.9}
              />
            </div>
            <span
              style={{
                fontSize: 10.5,
                fontWeight: active ? 800 : 600,
                fontFamily: "var(--font-display)",
              }}
            >
              {it.label}
            </span>
          </Link>
        )
      })}
    </div>
  )
}
