import type { components } from "./generated/schema"

type Schemas = components["schemas"]

export type MiniAppMeeting = Schemas["MiniAppMeeting"]
export type MiniAppMeetingsResponse = Schemas["MiniAppMeetingsResponse"]
export type MiniAppMeetingCreateRequest = Schemas["MiniAppMeetingCreateRequest"]
export type MiniAppMeetingUpdateRequest = Schemas["MiniAppMeetingUpdateRequest"]
export type MiniAppEmployee = Schemas["MiniAppEmployee"]
export type MiniAppEmployeesResponse = Schemas["MiniAppEmployeesResponse"]
export type MiniAppFreeSlot = Schemas["MiniAppFreeSlot"]
export type MiniAppFreeSlotsResponse = Schemas["MiniAppFreeSlotsResponse"]
export type MiniAppConflict = Schemas["MiniAppConflict"]
export type MiniAppOccurrenceConflicts = Schemas["MiniAppOccurrenceConflicts"]
export type MiniAppConflictsRequest = Schemas["MiniAppConflictsRequest"]
export type MiniAppFreeSlotsRequest = Schemas["MiniAppFreeSlotsRequest"]
export type MiniAppUserSettings = Schemas["MiniAppUserSettings"]
export type MiniAppUserSettingsPatch = Schemas["MiniAppUserSettingsPatch"]

export type WebMeeting = Schemas["Meeting"]
export type WebMeetingParticipant = Schemas["MeetingParticipant"]
export type WebMeetingCreateRequest = Schemas["WebMeetingCreateRequest"]
export type WebMeetingUpdateRequest = Schemas["WebMeetingUpdateRequest"]
