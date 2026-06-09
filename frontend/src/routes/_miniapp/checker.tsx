import { createFileRoute } from "@tanstack/react-router"
import { CheckerPage } from "@/features/checker/pages/checker-page"

export const Route = createFileRoute("/_miniapp/checker")({
  component: CheckerPage,
})
