import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
  isAxiosError,
} from "axios"

import { clearSession, getSession } from "~/shared/auth/session"
import { ApiError, type ApiErrorBody, type ApiFetchOptions } from "~/shared/api/types"

type RetryableConfig = InternalAxiosRequestConfig & { _retry?: boolean }

let reauthHandler: (() => Promise<string | null>) | null = null

export function setReauthHandler(handler: (() => Promise<string | null>) | null): void {
  reauthHandler = handler
}

function parseErrorMessage(status: number, body: unknown): { message: string; code?: string } {
  if (body && typeof body === "object") {
    const record = body as ApiErrorBody
    if (record.error?.message) {
      return { message: record.error.message, code: record.error.code }
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

function toApiError(error: unknown): ApiError {
  if (isAxiosError(error)) {
    const status = error.response?.status ?? 0
    const body = error.response?.data
    const { message, code } =
      status > 0 ? parseErrorMessage(status, body) : { message: error.message, code: error.code }
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

export const api: AxiosInstance = axios.create({
  baseURL: "",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json",
  },
  paramsSerializer: { indexes: null },
})

api.interceptors.request.use((config) => {
  const session = getSession()
  if (session?.token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${session.token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const axiosError = error as AxiosError
    const config = axiosError.config as RetryableConfig | undefined
    const isAuthCall = config?.url?.includes("/api/auth/miniapp")

    if (axiosError.response?.status === 401 && config && !config._retry && !isAuthCall) {
      config._retry = true
      if (reauthHandler) {
        const token = await reauthHandler()
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
          return api.request(config)
        }
      }
      clearSession()
    }
    return Promise.reject(toApiError(error))
  }
)

export async function apiFetch<T>(path: string, options: ApiFetchOptions = {}): Promise<T> {
  const config: AxiosRequestConfig = {
    url: path,
    method: options.method ?? (options.body !== undefined ? "POST" : "GET"),
    data: options.body,
    params: options.params,
    signal: options.signal,
  }
  const response = await api.request<T>(config)
  return response.data
}

export function isAbortError(error: unknown): boolean {
  if (!isAxiosError(error)) {
    return error instanceof DOMException && error.name === "AbortError"
  }
  return error.code === AxiosError.ERR_CANCELED || error.name === "CanceledError"
}
