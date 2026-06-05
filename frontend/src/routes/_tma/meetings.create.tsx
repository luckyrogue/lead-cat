import { createFileRoute } from "@tanstack/react-router"
import { CreateMeetingPage } from "@/features/meeting-create/pages/create-page"
import { myMeetingsQuery } from "@/features/meetings/queries"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/meetings/create")({
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: CreateMeetingPage,
})
