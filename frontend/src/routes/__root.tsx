import { createRootRoute, Outlet, useRouterState } from "@tanstack/react-router"
import { AuthGate } from "@/features/auth/auth-gate"
import { CatShell } from "@/widgets/cat-shell/cat-shell"

const LEGACY_PREFIXES = [
  "/workspaces",
  "/dashboard",
  "/scenarios",
  "/team",
  "/integrations",
  "/chat-link",
]

function RootLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const onLogin = pathname === "/login"
  const legacy = LEGACY_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`)
  )

  return (
    <AuthGate>
      {onLogin ? (
        <Outlet />
      ) : legacy ? (
        <CatShell>
          <Outlet />
        </CatShell>
      ) : (
        <Outlet />
      )}
    </AuthGate>
  )
}

export const Route = createRootRoute({
  component: RootLayout,
})
