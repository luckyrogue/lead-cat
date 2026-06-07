import { createFileRoute } from "@tanstack/react-router"
import { CreateMeetingPage } from "@/features/meeting-create/pages/create-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"

export const Route = createFileRoute("/_tma/meetings/create/$editId")({
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: CreateMeetingPage,
})
