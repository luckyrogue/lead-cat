import { useQuery } from "@tanstack/react-query"

import { getMeeting, listMeetings } from "~/entities/meeting/api"
import type { MeetingListFilter } from "~/entities/meeting/types"

export const meetingKeys = {
  list: (orgId: string) => ["orgs", orgId, "meetings"] as const,
  listFiltered: (orgId: string, filter: MeetingListFilter) =>
    ["orgs", orgId, "meetings", filter] as const,
  detail: (orgId: string, meetingId: string) =>
    ["orgs", orgId, "meetings", meetingId] as const,
}

export function useMeetings(
  orgId: string | null,
  filter: MeetingListFilter = {}
) {
  return useQuery({
    queryKey: meetingKeys.listFiltered(orgId ?? "", filter),
    queryFn: () => listMeetings(orgId as string, filter),
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
