import { createFileRoute } from "@tanstack/react-router"
import { HomePage } from "@/features/home/pages/home-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/")({
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("upcoming")),
  component: HomePage,
})
