import { useQuery } from "@tanstack/react-query"

import { getMeeting, listMeetings } from "~/entities/meeting/api"
import type { MeetingListFilter } from "~/entities/meeting/types"

export const meetingKeys = {
  all: (orgId: string) => ["orgs", orgId, "meetings"] as const,
  lists: (orgId: string) => ["orgs", orgId, "meetings", "list"] as const,
  listFiltered: (orgId: string, filter: MeetingListFilter) =>
    ["orgs", orgId, "meetings", "list", filter] as const,
  detail: (orgId: string, meetingId: string) =>
    ["orgs", orgId, "meetings", "detail", meetingId] as const,
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
