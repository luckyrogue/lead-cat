import { queryOptions, useMutation, useQuery } from "@tanstack/react-query"
import {
  fetchFreeSlots,
  searchEmployees,
  type FreeSlotsParams,
} from "@/features/meetings/api"
import { tmaKeys } from "@/shared/api/query-keys"
import { useTmaApp } from "@/shared/tma/context"

export const employeeSearchQuery = (q: string) =>
  queryOptions({
    queryKey: tmaKeys.employees(q),
    queryFn: ({ signal }) => searchEmployees(q, signal),
    enabled: q.trim().length > 0,
  })

export function useEmployeeSearch(q: string) {
  return useQuery(employeeSearchQuery(q))
}

export function useFreeSlots() {
  const { lang } = useTmaApp()
  return useMutation({
    mutationFn: (params: FreeSlotsParams) => fetchFreeSlots(params, lang),
  })
}
