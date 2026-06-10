import { createRoute } from "@tanstack/react-router";
import { Route as RootRoute } from "@/routes/__root";
import { DashboardPage } from "@/features/dashboard/pages/dashboard-page";

export const Route = createRoute({
  getParentRoute: () => RootRoute,
  path: "/",
  component: DashboardPage,
});
