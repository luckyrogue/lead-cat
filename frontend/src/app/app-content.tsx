import { RouterProvider } from "@tanstack/react-router"

import { MaintenanceScreen } from "@/components/maintenance-screen"
import { Toaster } from "@/components/ui/sonner"
import { isHealthy, useAppHealth } from "@/shared/api/health"

import { queryClient, router } from "@/app/router"

export function AppContent() {
  const healthQuery = useAppHealth()
  const isMaintenanceMode =
    healthQuery.isError ||
    (healthQuery.data !== undefined && !isHealthy(healthQuery.data))

  if (isMaintenanceMode) {
    return <MaintenanceScreen />
  }

  return (
    <>
      <RouterProvider router={router} context={{ queryClient }} />
      <Toaster richColors closeButton position="top-center" />
    </>
  )
}
