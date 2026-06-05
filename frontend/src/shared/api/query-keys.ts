import type { TmaMeetingsScope } from "@/shared/api/types"

export const tmaKeys = {
  all: ["tma"] as const,
  meetings: (scope: TmaMeetingsScope) =>
    [...tmaKeys.all, "meetings", scope] as const,
  schedule: (email: string, scope: TmaMeetingsScope) =>
    [...tmaKeys.all, "schedule", email, scope] as const,
  employees: (q: string) => [...tmaKeys.all, "employees", q] as const,
}

export const healthKeys = {
  all: ["health"] as const,
  status: () => [...healthKeys.all, "status"] as const,
}
