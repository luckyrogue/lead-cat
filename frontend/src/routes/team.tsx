import { createFileRoute } from "@tanstack/react-router"
import { TeamPage } from "@/features/team/team-page"

export const Route = createFileRoute("/team")({
  validateSearch: (s: Record<string, unknown>) => ({
    workspaceId: (s.workspaceId as string) || "",
  }),
  component: TeamPage,
})
