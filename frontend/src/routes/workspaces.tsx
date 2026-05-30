import { createFileRoute } from "@tanstack/react-router"
import { WorkspacesPage } from "@/features/workspace-list/workspaces-page"

export const Route = createFileRoute("/workspaces")({
  component: WorkspacesPage,
})
