import type {
  WebMeeting,
  WebMeetingCreateRequest,
  WebMeetingParticipant,
  WebMeetingUpdateRequest,
} from "@leadcat/api-client"

export type Meeting = WebMeeting
export type MeetingParticipant = WebMeetingParticipant
export type CreateMeetingInput = WebMeetingCreateRequest
export type UpdateMeetingInput = WebMeetingUpdateRequest

export type MeetingRecurrence =
  | "once"
  | "daily"
  | "weekly"
  | "monthly"
  | "custom"

export type MeetingScope = "this" | "whole"

export function isSeries(meeting: Meeting): boolean {
  return meeting.recurrence !== "once" || Boolean(meeting.series_id)
}
