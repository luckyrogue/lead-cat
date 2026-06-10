import { createFileRoute } from "@tanstack/react-router"
import { WebLayout } from "@/components/web-shell/web-layout"

export const Route = createFileRoute("/_web")({
  component: WebLayout,
})
