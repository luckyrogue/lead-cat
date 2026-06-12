import axios, { type AxiosInstance } from "axios"

export type CreateApiClientOptions = {
  withCredentials?: boolean
}

export function createApiClient(
  baseURL: string,
  options: CreateApiClientOptions = {}
): AxiosInstance {
  return axios.create({
    baseURL,
    withCredentials: options.withCredentials ?? true,
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    paramsSerializer: { indexes: null },
  })
}
