import { useQuery } from "@tanstack/react-query"

import { getMeeting, listMeetings } from "~/entities/meeting/api"

export const meetingKeys = {
  list: (orgId: string) => ["orgs", orgId, "meetings"] as const,
  detail: (orgId: string, meetingId: string) =>
    ["orgs", orgId, "meetings", meetingId] as const,
}

export function useMeetings(orgId: string | null) {
  return useQuery({
    queryKey: meetingKeys.list(orgId ?? ""),
    queryFn: () => listMeetings(orgId as string),
    enabled: Boolean(orgId),
  })
}

export function useMeeting(orgId: string, meetingId: string | null) {
  return useQuery({
    queryKey: meetingKeys.detail(orgId, meetingId ?? ""),
    queryFn: () => getMeeting(orgId, meetingId as string),
    enabled: Boolean(orgId) && Boolean(meetingId),
  })
}
