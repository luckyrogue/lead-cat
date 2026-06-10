import { useQuery } from "@tanstack/react-query"

import { api, toApiError } from "~/shared/api/client"
import type { Me } from "~/shared/auth/types"

export const meQueryKey = ["auth", "me"] as const

async function fetchMe(): Promise<Me | null> {
  try {
    const { data } = await api.get<Me>("/api/auth/web/me")
    return data
  } catch (error) {
    const apiError = toApiError(error)
    if (apiError.status === 401) {
      return null
    }
    throw apiError
  }
}

export function useMe() {
  return useQuery({
    queryKey: meQueryKey,
    queryFn: fetchMe,
    retry: false,
    staleTime: 60_000,
  })
}
