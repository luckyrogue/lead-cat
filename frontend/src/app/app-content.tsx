import { RouterProvider } from "@tanstack/react-router"

import { MaintenanceScreen } from "@/components/maintenance-screen"
import { Toaster } from "@/components/ui/sonner"
import { isHealthy, useAppHealth } from "@/shared/api/health"
import { WebAuthProvider } from "@/shared/web-auth/context"
import { getTelegramWebApp } from "@/shared/miniapp/telegram-env"
import { detectSurface } from "@/shared/lib/surface"

import { queryClient, router } from "@/app/router"

const surface = detectSurface(getTelegramWebApp())

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
    return <WebAuthProvider>{routerProvider}</WebAuthProvider>
  }

  return routerProvider
}
