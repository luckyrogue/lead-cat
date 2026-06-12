import { isAxiosError } from "axios"

export type ApiErrorBody = {
  code?: string
  message?: string
  error?: string | { code?: string; message?: string }
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: string
  readonly body?: unknown

  constructor(status: number, message: string, code?: string, body?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
    this.body = body
  }
}

function parseErrorMessage(status: number, body: unknown): { message: string; code?: string } {
  if (body && typeof body === "object") {
    const record = body as ApiErrorBody
    if (record.error && typeof record.error === "object" && record.error.message) {
      return { message: record.error.message, code: record.error.code }
    }
    if (typeof record.error === "string") {
      return { message: record.error, code: record.error }
    }
    if (record.message) {
      return { message: record.message, code: record.code }
    }
    if (record.code) {
      return { message: record.code, code: record.code }
    }
  }
  return { message: `Request failed (${status})` }
}

export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }
  if (isAxiosError(error)) {
    const status = error.response?.status ?? 0
    const body = error.response?.data
    const { message, code } =
      status > 0 ? parseErrorMessage(status, body) : { message: error.message, code: error.code }
    return new ApiError(status, message, code, body)
  }
  if (error instanceof Error) {
    return new ApiError(0, error.message)
  }
  return new ApiError(0, "Unknown error")
}

export function isAbortError(error: unknown): boolean {
  if (!isAxiosError(error)) {
    return error instanceof DOMException && error.name === "AbortError"
  }
  return error.code === "ERR_CANCELED" || error.name === "CanceledError"
}
