import { createFileRoute, redirect } from "@tanstack/react-router"
import { getSession } from "@/features/auth/session"
import { AdminPanelPage } from "@/features/profile/pages/admin-panel-page"
import { myMeetingsQuery } from "@/features/meetings/queries"
import { canAccessTmaRoute } from "@/shared/auth/route-access"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/profile/admin")({
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  beforeLoad: () => {
    const session = getSession()
    if (!canAccessTmaRoute(session?.user, "admin")) {
      throw redirect({ to: "/profile" })
    }
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: AdminPanelPage,
})
