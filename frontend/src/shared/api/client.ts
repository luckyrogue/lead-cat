import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
  isAxiosError,
} from "axios"

import { getSession } from "@/shared/auth/session"
import type { ApiErrorBody, ApiFetchOptions } from "@/shared/api/types"
import { ApiError } from "@/shared/api/types"

const AUTH_PATHS_WITHOUT_RETRY = ["/auth/tma"]

type RetryableAxiosRequestConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
}

export function getApiUrl(): string {
  const url = import.meta.env.VITE_API_URL
  if (typeof url === "string" && url.length > 0) {
    return url.replace(/\/$/, "")
  }
  return "/api"
}

/** @deprecated use getSession()?.accessToken */
export function getAuthToken(): string | null {
  return getSession()?.accessToken ?? null
}

/** Dev bootstrap only — prefer setSession via tmaLogin. */
export function setAuthToken(token: string | null) {
  if (token) {
    api.defaults.headers.common.Authorization = `Bearer ${token}`
  } else {
    delete api.defaults.headers.common.Authorization
  }
}

function parseErrorMessage(
  status: number,
  body: unknown
): { message: string; code?: string } {
  if (body && typeof body === "object") {
    const record = body as ApiErrorBody
    if (typeof record.message === "string" && record.message.length > 0) {
      return { message: record.message, code: record.code }
    }
    if (typeof record.code === "string" && record.code.length > 0) {
      return { message: record.code, code: record.code }
    }
    if (typeof record.detail === "string") {
      return { message: record.detail }
    }
    if (Array.isArray(record.detail)) {
      return { message: record.detail.map(String).join("; ") }
    }
  }
  return { message: `Request failed (${status})` }
}

export function toApiError(error: unknown): ApiError {
  if (isAxiosError(error)) {
    const status = error.response?.status ?? 0
    const body = error.response?.data
    const { message, code } =
      status > 0
        ? parseErrorMessage(status, body)
        : { message: error.message, code: error.code }
    return new ApiError(status, message, code, body)
  }
  if (error instanceof ApiError) {
    return error
  }
  if (error instanceof Error) {
    return new ApiError(0, error.message)
  }
  return new ApiError(0, "Unknown error")
}

function isAuthPathWithoutRetry(url: string | undefined): boolean {
  if (!url) return false
  return AUTH_PATHS_WITHOUT_RETRY.some((path) => url.includes(path))
}

function isTmaApiPath(url: string | undefined): boolean {
  if (!url) return false
  return url.includes("/tma/")
}

export const api: AxiosInstance = axios.create({
  baseURL: getApiUrl(),
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json",
  },
  paramsSerializer: {
    indexes: null,
  },
})

api.interceptors.request.use((config) => {
  const session = getSession()
  if (session?.accessToken && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${session.accessToken}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (isAbortError(error)) {
      return Promise.reject(error)
    }

    const axiosError = error as AxiosError
    const config = axiosError.config as RetryableAxiosRequestConfig | undefined

    if (
      axiosError.response?.status === 401 &&
      config &&
      !config._retry &&
      isTmaApiPath(config.url) &&
      !isAuthPathWithoutRetry(config.url)
    ) {
      config._retry = true

      const { refreshTmaSessionIfNeeded } =
        await import("@/shared/auth/refresh-session")
      const token = await refreshTmaSessionIfNeeded({ force: true })

      if (token) {
        config.headers.Authorization = `Bearer ${token}`
        return api.request(config)
      }
    }

    return Promise.reject(toApiError(error))
  }
)

export async function apiFetch<T>(
  path: string,
  options: ApiFetchOptions = {}
): Promise<T> {
  const headers: Record<string, string> = {}

  if (options.accessToken) {
    headers.Authorization = `Bearer ${options.accessToken}`
  }

  const config: AxiosRequestConfig = {
    url: path,
    method: options.method ?? (options.body !== undefined ? "POST" : "GET"),
    data: options.body,
    params: options.params,
    headers: Object.keys(headers).length > 0 ? headers : undefined,
    signal: options.signal,
  }

  const response = await api.request<T>(config)
  return response.data
}

export function createRequestAbort() {
  const controller = new AbortController()
  return { signal: controller.signal, abort: () => controller.abort() }
}

export function isAbortError(error: unknown): boolean {
  if (!isAxiosError(error)) {
    return error instanceof DOMException && error.name === "AbortError"
  }
  return (
    error.code === AxiosError.ERR_CANCELED || error.name === "CanceledError"
  )
}

export { isAxiosError }
