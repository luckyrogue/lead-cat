import { api } from "@/shared/api/client"

export type Workspace = { id: string; slug: string; name: string }

export async function fetchWorkspaces(): Promise<Workspace[]> {
  const { data } = await api.get<Workspace[]>("/workspaces")
  return data
}

export async function createWorkspace(
  name: string,
  slug: string
): Promise<Workspace> {
  const { data } = await api.post<Workspace>("/workspaces", { name, slug })
  return data
}
