import { ApiError, createApiClient, toApiError } from "@leadcat/api-client"
import { isAxiosError, type AxiosInstance } from "axios"

import { getActiveOrgId } from "~/shared/api/active-org"
import { toastApiError } from "~/shared/lib/toast"

const MUTATION_METHODS = new Set(["post", "put", "patch", "delete"])

let csrfWarningLogged = false

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

function isAuthPath(url: string | undefined): boolean {
  return typeof url === "string" && url.includes("/api/auth/")
}

function isCsrfExemptPath(url: string | undefined): boolean {
  return typeof url === "string" && url.includes("/api/auth/web/magic/")
}

export function prepareMutationCsrf(
  method: string,
  csrf: string | null,
  url: string,
  dev = import.meta.env.DEV
): { header?: string; warn?: boolean } {
  if (!MUTATION_METHODS.has(method.toLowerCase())) {
    return {}
  }
  if (csrf) {
    return { header: csrf }
  }
  // Only unauthenticated magic-link bootstrap endpoints have no session CSRF cookie yet — allow them.
  if (isCsrfExemptPath(url)) {
    return {}
  }
  if (dev) {
    return { warn: true }
  }
  throw new Error("missing_csrf_token")
}

function isCsrfError(apiError: ApiError): boolean {
  return apiError.code === "csrf" || apiError.message === "csrf"
}

export const api: AxiosInstance = createApiClient(resolveBaseUrl())

api.interceptors.request.use((config) => {
  const method = (config.method ?? "get").toLowerCase()
  const csrfResult = prepareMutationCsrf(
    method,
    readCsrfCookie(),
    config.url ?? ""
  )
  if (csrfResult.header) {
    config.headers.set("X-CSRF-Token", csrfResult.header)
  } else if (csrfResult.warn && !csrfWarningLogged) {
    console.warn("[api] mutation without lc_csrf cookie")
    csrfWarningLogged = true
  }
  const orgId = getActiveOrgId()
  if (orgId) {
    config.headers.set("X-Org-Id", orgId)
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const apiError = toApiError(error)

    if (apiError.status === 403 && isCsrfError(apiError)) {
      toastApiError(apiError, "errors.codes.csrf")
    }

    if (apiError.status === 401 && !isAuthPath(error.config?.url)) {
      if (typeof window !== "undefined") {
        window.location.href = "/login"
      }
    }

    return Promise.reject(apiError)
  }
)

export { ApiError, isAxiosError, toApiError }
