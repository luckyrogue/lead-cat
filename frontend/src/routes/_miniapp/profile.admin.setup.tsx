import { createFileRoute, redirect } from "@tanstack/react-router"
import { AdminSetupPage } from "@/features/admin-setup/pages/admin-setup-page"
import { getSession } from "@/shared/auth/session"
import { canAccessMiniAppRoute } from "@/shared/auth/route-access"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_miniapp/profile/admin/setup")({
  shouldReload: shouldReloadExceptSearch,
  beforeLoad: () => {
    const session = getSession()
    if (!canAccessMiniAppRoute(session?.user, "admin")) {
      throw redirect({ to: "/profile" })
    }
  },
  component: AdminSetupPage,
})
