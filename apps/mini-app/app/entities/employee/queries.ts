import { queryOptions } from "@tanstack/react-query"

import { searchEmployees } from "~/entities/employee/api"
import { employeeKeys } from "~/shared/api/query-keys"

export function employeeSearchQuery(q: string) {
  return queryOptions({
    queryKey: employeeKeys.search(q),
    queryFn: () => searchEmployees(q),
    enabled: q.trim().length > 0,
    staleTime: 60_000,
  })
}
