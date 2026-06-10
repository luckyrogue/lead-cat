import { createApiClient } from "@leadcat/api-client"
import { type AxiosInstance, isAxiosError } from "axios"

import { getActiveOrgId } from "~/shared/api/active-org"

const MUTATION_METHODS = new Set(["post", "put", "patch", "delete"])

function readCsrfCookie(): string | null {
  if (typeof document === "undefined") {
    return null
  }
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith("lc_csrf="))
  return match ? decodeURIComponent(match.slice("lc_csrf=".length)) : null
}

function resolveBaseUrl(): string {
  const url = import.meta.env.VITE_API_URL
  return typeof url === "string" && url.length > 0 ? url.replace(/\/$/, "") : ""
}

export const api: AxiosInstance = createApiClient(resolveBaseUrl())

api.interceptors.request.use((config) => {
  const method = (config.method ?? "get").toLowerCase()
  if (MUTATION_METHODS.has(method)) {
    const csrf = readCsrfCookie()
    if (csrf) {
      config.headers.set("X-CSRF-Token", csrf)
    }
  }
  const orgId = getActiveOrgId()
  if (orgId) {
    config.headers.set("X-Org-Id", orgId)
  }
  return config
})

export class ApiError extends Error {
  readonly status: number
  readonly code: string | undefined

  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

export function toApiError(error: unknown): ApiError {
  if (isAxiosError(error)) {
    const status = error.response?.status ?? 0
    const body = error.response?.data as { error?: string } | undefined
    const code = typeof body?.error === "string" ? body.error : undefined
    return new ApiError(status, code ?? error.message, code)
  }
  if (error instanceof Error) {
    return new ApiError(0, error.message)
  }
  return new ApiError(0, "Unknown error")
}

export { isAxiosError }
