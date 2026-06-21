import type { QueryClient } from "@tanstack/react-query"

import { meQueryKey } from "~/shared/auth/use-me"

export function invalidateMeExact(queryClient: QueryClient) {
  return queryClient.invalidateQueries({ queryKey: meQueryKey, exact: true })
}
