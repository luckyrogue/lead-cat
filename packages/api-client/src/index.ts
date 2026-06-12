export { createApiClient } from "./client";
export { ApiError, toApiError, isAbortError } from "./errors";
export type { ApiErrorBody } from "./errors";
export { apiFetch } from "./fetch";
export type { ApiFetchOptions } from "./fetch";
export type { paths, components } from "./generated/schema";
export type {
  MiniAppMeeting,
  MiniAppMeetingsResponse,
  MiniAppMeetingCreateRequest,
  MiniAppMeetingUpdateRequest,
  MiniAppEmployee,
  MiniAppEmployeesResponse,
  MiniAppFreeSlot,
  MiniAppFreeSlotsResponse,
  MiniAppConflict,
  MiniAppOccurrenceConflicts,
  MiniAppConflictsRequest,
  MiniAppFreeSlotsRequest,
  MiniAppUserSettings,
  MiniAppUserSettingsPatch,
  WebMeeting,
  WebMeetingParticipant,
  WebMeetingCreateRequest,
  WebMeetingUpdateRequest,
} from "./types";
