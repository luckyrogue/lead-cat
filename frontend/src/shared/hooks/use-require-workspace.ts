import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useEffect } from "react";
import { isWorkspaceRoute, useWorkspaceId } from "@/shared/hooks/use-workspace-id";

export function useRequireWorkspace(): boolean {
  const workspaceId = useWorkspaceId();
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const fromUrl = useRouterState({
    select: (s) => (s.location.search as { workspaceId?: string }).workspaceId ?? "",
  });

  useEffect(() => {
    if (!workspaceId) {
      navigate({ to: "/workspaces" });
      return;
    }
    if (!fromUrl && isWorkspaceRoute(pathname)) {
      navigate({ to: pathname, search: { workspaceId }, replace: true });
    }
  }, [workspaceId, fromUrl, pathname, navigate]);

  return !!workspaceId;
}
