import { createFileRoute, redirect } from "@tanstack/react-router"
import { getSession } from "@/shared/auth/session"
import { AdminPanelPage } from "@/features/profile/pages/admin-panel-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"
import { canAccessMiniAppRoute } from "@/shared/auth/route-access"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_miniapp/profile/admin")({
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  beforeLoad: () => {
    const session = getSession()
    if (!canAccessMiniAppRoute(session?.user, "admin")) {
      throw redirect({ to: "/profile" })
    }
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: AdminPanelPage,
})
