import { queryOptions } from "@tanstack/react-query"

import { fetchMyMeetings, fetchSchedule } from "~/entities/meeting/api"
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

export function scheduleQuery(email: string, scope: MeetingScope) {
  return queryOptions({
    queryKey: scheduleKeys.byEmail(email, scope),
    queryFn: () => fetchSchedule(email, scope),
    enabled: email.length > 0,
  })
}
