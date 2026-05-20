import { createFileRoute } from "@tanstack/react-router";
import { DashboardPage } from "@/features/dashboard/dashboard-page";

export const Route = createFileRoute("/dashboard")({
  validateSearch: (s: Record<string, unknown>) => ({
    workspaceId: (s.workspaceId as string) || "",
  }),
  component: DashboardPage,
});
