import { api } from "@/shared/api/client";

export type Scenario = {
  id: string;
  name: string;
  enabled: boolean;
  definition: { nodes: unknown[]; edges: unknown[] };
};

export async function createScenario(
  workspaceId: string,
  body: { name: string; definition: unknown },
): Promise<Scenario> {
  const { data } = await api.post<Scenario>(`/workspaces/${workspaceId}/scenarios`, body);
  return data;
}

export async function fetchScenarios(workspaceId: string): Promise<Scenario[]> {
  const { data } = await api.get<Scenario[]>(`/workspaces/${workspaceId}/scenarios`);
  return data;
}

export async function fetchScenario(workspaceId: string, scenarioId: string): Promise<Scenario> {
  const { data } = await api.get<Scenario>(`/workspaces/${workspaceId}/scenarios/${scenarioId}`);
  return data;
}

export async function updateScenario(
  workspaceId: string,
  scenarioId: string,
  body: { name?: string; enabled?: boolean; definition?: unknown },
): Promise<Scenario> {
  const { data } = await api.patch<Scenario>(`/workspaces/${workspaceId}/scenarios/${scenarioId}`, body);
  return data;
}

export async function runScenario(workspaceId: string, scenarioId: string): Promise<void> {
  await api.post(`/workspaces/${workspaceId}/scenarios/${scenarioId}/run`);
}
