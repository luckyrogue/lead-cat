import { useEffect, type ReactNode } from "react"
import { RouterProvider } from "@tanstack/react-router"

import { MaintenanceScreen } from "@/components/maintenance-screen"
import { Toaster } from "@/components/ui/sonner"
import { isHealthy, useAppHealth } from "@/shared/api/health"
import { WebAuthProvider } from "@/shared/web-auth/context"
import { getTelegramWebApp } from "@/shared/miniapp/telegram-env"
import { detectSurface } from "@/shared/lib/surface"

import { queryClient, router } from "@/app/router"

const surface = detectSurface(getTelegramWebApp())

/**
 * For web users landing on `/`, the router would otherwise match the parked
 * Telegram MiniApp index route (fullPath `/`). This component fires once on
 * mount and navigates to `/dashboard`; the WebLayout guard then takes over
 * (anonymous → /login, authed+0orgs → /onboarding, else stays on /dashboard).
 */
function WebRootRedirect({ children }: { children: ReactNode }) {
  useEffect(() => {
    if (window.location.pathname === "/") {
      void router.navigate({ to: "/dashboard", replace: true })
    }
  }, [])

  return <>{children}</>
}

export function AppContent() {
  const healthQuery = useAppHealth()
  const isMaintenanceMode =
    healthQuery.isError ||
    (healthQuery.data !== undefined && !isHealthy(healthQuery.data))

  if (isMaintenanceMode) {
    return <MaintenanceScreen />
  }

  const routerProvider = (
    <>
      <RouterProvider router={router} context={{ queryClient }} />
      <Toaster richColors closeButton position="top-center" />
    </>
  )

  if (surface === "web") {
    return (
      <WebAuthProvider>
        <WebRootRedirect>{routerProvider}</WebRootRedirect>
      </WebAuthProvider>
    )
  }

  return routerProvider
}
