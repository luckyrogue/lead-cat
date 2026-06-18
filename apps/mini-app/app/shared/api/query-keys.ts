export type MeetingScope = "upcoming" | "past" | "all"

export const meetingKeys = {
  all: ["meetings"] as const,
  lists: () => [...meetingKeys.all, "list"] as const,
  list: (scope: MeetingScope) => [...meetingKeys.lists(), scope] as const,
  details: () => [...meetingKeys.all, "detail"] as const,
  detail: (id: string) => [...meetingKeys.details(), id] as const,
}

export const employeeKeys = {
  all: ["employees"] as const,
  search: (q: string) => [...employeeKeys.all, "search", q] as const,
}

export const scheduleKeys = {
  all: ["schedule"] as const,
  byEmail: (email: string, scope: MeetingScope) =>
    [...scheduleKeys.all, email, scope] as const,
}

export const settingsKeys = {
  all: ["settings"] as const,
}

export const meKeys = {
  all: ["me"] as const,
}

export const calendarConnectionKeys = {
  all: ["calendar-connections"] as const,
}
