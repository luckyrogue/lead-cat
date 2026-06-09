import { createFileRoute } from "@tanstack/react-router"
import { MiniAppLayout } from "@/components/miniapp-shell/miniapp-layout"

export const Route = createFileRoute("/_miniapp")({
  component: MiniAppLayout,
})
