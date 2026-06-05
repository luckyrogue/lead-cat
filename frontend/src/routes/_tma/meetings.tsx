import { createFileRoute } from "@tanstack/react-router"
import { z } from "zod"
import { MeetingsListPage } from "@/features/meetings/pages/meetings-list-page"
import { myMeetingsQuery } from "@/features/meetings/queries"
import { shouldReloadExceptSearch } from "@/shared/lib/route-revalidation"

const meetingsSearchSchema = z.object({
  scope: z.enum(["upcoming", "past", "all"]).optional().catch("upcoming"),
  q: z.string().optional().catch(""),
  page: z.coerce.number().optional().catch(1),
  success: z.string().optional(),
})

export type MeetingsSearch = z.infer<typeof meetingsSearchSchema>

export const Route = createFileRoute("/_tma/meetings")({
  validateSearch: (search) => meetingsSearchSchema.parse(search),
  shouldReload: shouldReloadExceptSearch,
  loaderDeps: () => ({}),
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: MeetingsListPage,
})
