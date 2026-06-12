import {
  apiFetch as sharedApiFetch,
  ApiError,
  createApiClient,
  isAbortError,
  toApiError,
  type ApiFetchOptions,
} from "@leadcat/api-client"
import type { AxiosError, InternalAxiosRequestConfig } from "axios"

import { clearSession, getSession } from "~/shared/auth/session"

type RetryableConfig = InternalAxiosRequestConfig & { _retry?: boolean }

let reauthHandler: (() => Promise<string | null>) | null = null

export function setReauthHandler(handler: (() => Promise<string | null>) | null): void {
  reauthHandler = handler
}

export const api = createApiClient("", { withCredentials: false })

api.interceptors.request.use((config) => {
  const session = getSession()
  if (session?.token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${session.token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = error.config as RetryableConfig | undefined
    const isAuthCall = config?.url?.includes("/api/auth/miniapp")

    if (error.response?.status === 401 && config && !config._retry && !isAuthCall) {
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
  return sharedApiFetch<T>(api, path, options)
}

export { ApiError, isAbortError }
