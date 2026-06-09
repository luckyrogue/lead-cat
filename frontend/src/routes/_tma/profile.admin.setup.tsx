import { createFileRoute, redirect } from "@tanstack/react-router"
import { AdminSetupPage } from "@/features/admin-setup/pages/admin-setup-page"
import { getSession } from "@/shared/auth/session"
import { canAccessTmaRoute } from "@/shared/auth/route-access"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/profile/admin/setup")({
  shouldReload: shouldReloadExceptSearch,
  beforeLoad: () => {
    const session = getSession()
    if (!canAccessTmaRoute(session?.user, "admin")) {
      throw redirect({ to: "/profile" })
    }
  },
  component: AdminSetupPage,
})
