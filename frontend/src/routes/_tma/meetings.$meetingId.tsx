import { createFileRoute } from "@tanstack/react-router"
import { MeetingsListPage } from "@/features/meetings/pages/meetings-list-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"
import { Route as meetingsRoute } from "./meetings"

export const Route = createFileRoute("/_tma/meetings/$meetingId")({
  validateSearch: meetingsRoute.options.validateSearch,
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: MeetingsListPage,
})
