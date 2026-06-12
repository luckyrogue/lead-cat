import type {
  MiniAppConflict,
  MiniAppFreeSlot,
  MiniAppMeeting,
  MiniAppMeetingCreateRequest,
  MiniAppMeetingUpdateRequest,
  MiniAppOccurrenceConflicts,
} from "@leadcat/api-client"

export type Meeting = MiniAppMeeting
export type CreateMeetingInput = MiniAppMeetingCreateRequest
export type UpdateMeetingInput = MiniAppMeetingUpdateRequest
export type Conflict = MiniAppConflict
export type OccurrenceConflicts = MiniAppOccurrenceConflicts
export type FreeSlot = MiniAppFreeSlot

export type MeetingRecurrence =
  | "once"
  | "daily"
  | "weekly"
  | "monthly"
  | "custom"

export type MeetingMutationScope = "this" | "whole"

export function isSeriesMeeting(meeting: Pick<Meeting, "rec">): boolean {
  return meeting.rec !== "" && meeting.rec !== "once"
}
