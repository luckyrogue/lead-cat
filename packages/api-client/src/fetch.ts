import type { AxiosInstance, AxiosRequestConfig } from "axios"

export type ApiFetchOptions = {
  method?: "GET" | "POST" | "PATCH" | "DELETE"
  body?: unknown
  params?: Record<string, unknown>
  signal?: AbortSignal
}

export async function apiFetch<T>(
  client: AxiosInstance,
  path: string,
  options: ApiFetchOptions = {}
): Promise<T> {
  const config: AxiosRequestConfig = {
    url: path,
    method: options.method ?? (options.body !== undefined ? "POST" : "GET"),
    data: options.body,
    params: options.params,
    signal: options.signal,
  }
  const response = await client.request<T>(config)
  return response.data
}
