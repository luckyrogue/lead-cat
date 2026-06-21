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

export type Org = Schemas["Org"]
export type OrgMember = Schemas["OrgMember"]
export type OrgInvite = Schemas["OrgInvite"]

export type BookingEventType = Schemas["BookingEventType"]
export type BookingEventTypeInput = Schemas["BookingEventTypeInput"]
export type BookingEventTypesResponse = Schemas["BookingEventTypesResponse"]
export type PublicBookingView = Schemas["PublicBookingView"]
export type BookingEventView = Schemas["BookingEventView"]
export type BookingSlot = Schemas["BookingSlot"]
export type PublicBookingSubmitRequest = Schemas["PublicBookingSubmitRequest"]
export type BookingConfirmation = Schemas["BookingConfirmation"]

export type CalendarConnectionView = Schemas["CalendarConnectionView"]

export type MyInviteView = Schemas["MyInviteView"]
export type JoinRequestView = Schemas["JoinRequestView"]
export type JoinRequestAdminView = Schemas["JoinRequestAdminView"]
export type JoinRequestCreateResponse = Schemas["JoinRequestCreateResponse"]
