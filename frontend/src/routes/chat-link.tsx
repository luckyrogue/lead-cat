import { createFileRoute } from "@tanstack/react-router"
import { ChatLinkPage } from "@/features/chat-link/chat-link-page"

export const Route = createFileRoute("/chat-link")({
  validateSearch: (s: Record<string, unknown>) => ({
    workspaceId: (s.workspaceId as string) || "",
  }),
  component: ChatLinkPage,
})
