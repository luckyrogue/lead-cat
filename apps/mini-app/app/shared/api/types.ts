export type ApiErrorBody = {
  code?: string
  message?: string
  error?: { code?: string; message?: string }
}

export class ApiError extends Error {
  status: number
  code?: string
  body?: unknown

  constructor(status: number, message: string, code?: string, body?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
    this.body = body
  }
}

export type ApiFetchOptions = {
  method?: "GET" | "POST" | "PATCH" | "DELETE"
  body?: unknown
  params?: Record<string, unknown>
  signal?: AbortSignal
}
