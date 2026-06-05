import { createFileRoute } from "@tanstack/react-router"
import { TmaLayout } from "@/components/tma-shell/tma-layout"

export const Route = createFileRoute("/_tma")({
  component: TmaLayout,
})
