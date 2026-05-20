import { createFileRoute } from "@tanstack/react-router";
import { IntegrationsPage } from "@/features/integrations/integrations-page";

export const Route = createFileRoute("/integrations")({
  validateSearch: (s: Record<string, unknown>) => ({
    workspaceId: (s.workspaceId as string) || "",
  }),
  component: IntegrationsPage,
});
