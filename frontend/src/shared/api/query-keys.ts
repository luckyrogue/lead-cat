import type { MiniAppMeetingsScope } from "@/shared/api/types"

export const miniappKeys = {
  all: ["miniapp"] as const,
  meetings: (scope: MiniAppMeetingsScope) =>
    [...miniappKeys.all, "meetings", scope] as const,
  schedule: (email: string, scope: MiniAppMeetingsScope) =>
    [...miniappKeys.all, "schedule", email, scope] as const,
  employees: (q: string) => [...miniappKeys.all, "employees", q] as const,
}

export const healthKeys = {
  all: ["health"] as const,
  status: () => [...healthKeys.all, "status"] as const,
}
