import { apiFetch } from "~/shared/api/client"
import type { AuthUser, SessionData } from "~/shared/auth/session"

type AuthResponse = {
  token: string
  user: AuthUser
}

export async function authenticate(initData: string): Promise<SessionData> {
  const res = await apiFetch<AuthResponse>("/api/auth/miniapp", {
    method: "POST",
    body: { init_data: initData },
  })
  return { token: res.token, user: res.user }
}

export async function fetchMe(): Promise<AuthUser> {
  return apiFetch<AuthUser>("/api/miniapp/me")
}
