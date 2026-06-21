import { queryOptions } from "@tanstack/react-query"

import {
  fetchMeetingById,
  fetchMyMeetings,
  fetchSchedule,
} from "~/entities/meeting/api"
import {
  meetingKeys,
  scheduleKeys,
  type MeetingScope,
} from "~/shared/api/query-keys"

export function myMeetingsQuery(scope: MeetingScope) {
  return queryOptions({
    queryKey: meetingKeys.list(scope),
    queryFn: () => fetchMyMeetings(scope),
  })
}

export function meetingDetailQuery(id: string) {
  return queryOptions({
    queryKey: meetingKeys.detail(id),
    queryFn: () => fetchMeetingById(id),
    enabled: id.length > 0,
  })
}

export function scheduleQuery(email: string, scope: MeetingScope) {
  return queryOptions({
    queryKey: scheduleKeys.byEmail(email, scope),
    queryFn: () => fetchSchedule(email, scope),
    enabled: email.length > 0,
  })
}
