import { useRouterState } from "@tanstack/react-router";
import { useEffect, useMemo } from "react";

export const WORKSPACE_STORAGE_KEY = "lead-cat:workspace-id";

const workspaceRoutes = new Set([
  "/dashboard",
  "/scenarios",
  "/team",
  "/integrations",
  "/chat-link",
]);

export function isWorkspaceRoute(pathname: string): boolean {
  return workspaceRoutes.has(pathname);
}

/** workspaceId from URL search, with last-used fallback for shell navigation */
export function useWorkspaceId(): string {
  const fromUrl = useRouterState({
    select: (s) => (s.location.search as { workspaceId?: string }).workspaceId ?? "",
  });

  useEffect(() => {
    if (fromUrl) {
      setStoredWorkspaceId(fromUrl);
    }
  }, [fromUrl]);

  return useMemo(() => {
    if (fromUrl) return fromUrl;
    try {
      return sessionStorage.getItem(WORKSPACE_STORAGE_KEY) ?? "";
    } catch {
      return "";
    }
  }, [fromUrl]);
}

export function setStoredWorkspaceId(id: string) {
  try {
    sessionStorage.setItem(WORKSPACE_STORAGE_KEY, id);
  } catch {
    /* ignore */
  }
}
