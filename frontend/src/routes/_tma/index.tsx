import { createFileRoute } from "@tanstack/react-router"
import { HomePage } from "@/features/home/pages/home-page"
import { myMeetingsQuery } from "@/features/meetings/queries"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/")({
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: HomePage,
})
