import { queryOptions, useQuery } from "@tanstack/react-query"
import { searchEmployees } from "@/entities/employee/api"
import { miniappKeys } from "@/shared/api/query-keys"

export const employeeSearchQuery = (q: string) =>
  queryOptions({
    queryKey: miniappKeys.employees(q),
    queryFn: ({ signal }) => searchEmployees(q, signal),
    enabled: q.trim().length > 0,
  })

export function useEmployeeSearch(q: string) {
  return useQuery(employeeSearchQuery(q))
}
