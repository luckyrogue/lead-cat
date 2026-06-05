import { createFileRoute } from "@tanstack/react-router"
import { AutoPage } from "@/features/auto/pages/auto-page"

export const Route = createFileRoute("/_tma/auto")({
  component: AutoPage,
})
