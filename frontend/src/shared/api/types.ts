import type { operations, components } from "@/shared/api/generated/schema"
import type { AxiosRequestConfig } from "axios"

export type TmaMeetingsScope = "upcoming" | "past" | "all"

export type ApiFetchOptions = {
  method?: AxiosRequestConfig["method"]
  body?: unknown
  accessToken?: string
  signal?: AbortSignal
  params?: Record<string, unknown>
}

export type ApiErrorBody = {
  error?: string
  message?: string
  code?: string
  detail?: string | ApiErrorBody | ApiErrorBody[]
}

export type TmaUserDto = components["schemas"]["TmaUser"]
export type TmaMeetingDto = components["schemas"]["TmaMeeting"]
export type TmaEmployeeDto = components["schemas"]["TmaEmployee"]
export type TmaFreeSlotDto = components["schemas"]["TmaFreeSlot"]

export type TmaAuthResponse =
  operations["tma_auth_post"]["responses"][200]["content"]["application/json"]

export class ApiError extends Error {
  readonly status: number
  readonly code: string | undefined
  readonly body: unknown

  constructor(status: number, message: string, code?: string, body?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
    this.body = body
  }
}
