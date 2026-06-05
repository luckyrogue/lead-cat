import { queryOptions, useQuery } from "@tanstack/react-query"

import { apiFetch } from "@/shared/api/client"
import type { operations } from "@/shared/api/generated/schema"
import { healthKeys } from "@/shared/api/query-keys"

const HEALTH_POLL_INTERVAL_MS = 30_000

export type HealthResponse =
  operations["health_health_get"]["responses"][200]["content"]["application/json"]

export function isHealthy(data: HealthResponse): boolean {
  return data.postgres === "ok" && data.redis === "ok"
}

export async function getHealthStatus(): Promise<HealthResponse> {
  return apiFetch<HealthResponse>("/health")
}

export function healthStatusQueryOptions() {
  return queryOptions({
    queryKey: healthKeys.status(),
    queryFn: getHealthStatus,
    retry: false,
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
    refetchOnWindowFocus: true,
  })
}

export function useAppHealth() {
  return useQuery(healthStatusQueryOptions())
}
