import { createFileRoute } from "@tanstack/react-router"
import { ColleagueSchedulePage } from "@/features/profile/pages/colleague-schedule-page"

export const Route = createFileRoute("/_miniapp/profile/colleague")({
  component: ColleagueSchedulePage,
})
