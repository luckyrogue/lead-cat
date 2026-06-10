import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { createOrg, getMe, logout } from "./api"

export const webAuthKeys = {
  me: () => ["web", "me"] as const,
}

export function useMe() {
  return useQuery({
    queryKey: webAuthKeys.me(),
    queryFn: getMe,
    retry: false,
  })
}

export function useCreateOrg() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => createOrg(name),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: webAuthKeys.me() })
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: logout,
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: webAuthKeys.me() })
    },
  })
}
