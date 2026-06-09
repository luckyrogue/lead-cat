import { createFileRoute } from "@tanstack/react-router"
import { CreateMeetingPage } from "@/features/meeting-create/pages/create-page"
import { myMeetingsQuery } from "@/entities/meeting/queries"

type EditSearch = {
  scope?: "this" | "whole"
}

export const Route = createFileRoute("/_miniapp/meetings/create/$editId")({
  validateSearch: (search: Record<string, unknown>): EditSearch => {
    const scope = search.scope === "whole" ? "whole" : "this"
    return { scope }
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(myMeetingsQuery("all")),
  component: CreateMeetingPage,
})
