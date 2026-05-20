import { useRouterState } from "@tanstack/react-router";
import { useEffect, useMemo } from "react";

const STORAGE_KEY = "lead-cat:workspace-id";

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
      sessionStorage.setItem(STORAGE_KEY, fromUrl);
    }
  }, [fromUrl]);

  return useMemo(() => {
    if (fromUrl) return fromUrl;
    try {
      return sessionStorage.getItem(STORAGE_KEY) ?? "";
    } catch {
      return "";
    }
  }, [fromUrl]);
}
