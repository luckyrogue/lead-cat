import axios, { type AxiosInstance } from "axios"
import { getApiUrl } from "@/shared/api/client"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type WebUser = {
  id: string
  email: string
  avatar_url: string
  auth_method: string
}

export type WebOrg = {
  id: string
  name: string
  slug: string
}

// ---------------------------------------------------------------------------
// Cookie helpers
// ---------------------------------------------------------------------------

export function readCookie(name: string): string {
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith(`${name}=`))
  return match ? decodeURIComponent(match.split("=")[1] ?? "") : ""
}

export const ACTIVE_ORG_KEY = "lc.web.activeOrg"

function readActiveOrgId(): string {
  try {
    return localStorage.getItem(ACTIVE_ORG_KEY) ?? ""
  } catch {
    return ""
  }
}

// ---------------------------------------------------------------------------
// Axios instance — cookie-based, no Bearer token
// ---------------------------------------------------------------------------

export const webApi: AxiosInstance = axios.create({
  baseURL: getApiUrl(),
  withCredentials: true,
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json",
  },
})

// Attach CSRF token for mutating requests and X-Org-Id on every request
webApi.interceptors.request.use((config) => {
  const method = (config.method ?? "GET").toUpperCase()
  const mutating = ["POST", "PUT", "PATCH", "DELETE"].includes(method)
  if (mutating) {
    config.headers["X-CSRF-Token"] = readCookie("lc_csrf")
  }
  const orgId = readActiveOrgId()
  if (orgId) {
    config.headers["X-Org-Id"] = orgId
  }
  return config
})

// ---------------------------------------------------------------------------
// API functions
// ---------------------------------------------------------------------------

export async function getMe(): Promise<{ user: WebUser; organizations: WebOrg[] }> {
  const res = await webApi.get<{ user: WebUser; organizations: WebOrg[] }>("/auth/web/me")
  return res.data
}

export async function logout(): Promise<void> {
  await webApi.post("/auth/web/logout")
}

export async function requestMagicLink(email: string): Promise<void> {
  await webApi.post("/auth/web/magic/request", { email })
}

export async function createOrg(name: string): Promise<WebOrg> {
  const res = await webApi.post<WebOrg>("/orgs", { name })
  return res.data
}

export async function listMyOrgs(): Promise<WebOrg[]> {
  const res = await webApi.get<{ organizations: WebOrg[] }>("/orgs")
  return res.data.organizations
}

/** Returns the absolute URL to start an SSO flow (full-page redirect, not fetch). */
export function ssoStartUrl(provider: "google" | "microsoft"): string {
  return `${getApiUrl()}/auth/web/${provider}/start`
}
