import { createFileRoute } from "@tanstack/react-router"
import { TmaApp } from "@/features/tma/tma-app"

export const Route = createFileRoute("/")({
  component: TmaApp,
})
