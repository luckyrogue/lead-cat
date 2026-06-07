import { createFileRoute } from "@tanstack/react-router"
import { MeetingsListPage } from "@/features/meetings/pages/meetings-list-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"
import { meetingsSearchSchema } from "@/features/meetings/search-schema"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

export const Route = createFileRoute("/_tma/meetings")({
  validateSearch: (search) => meetingsSearchSchema.parse(search),
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: MeetingsListPage,
})
