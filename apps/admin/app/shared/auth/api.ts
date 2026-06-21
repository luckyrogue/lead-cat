import { api } from "~/shared/api/client"

export async function requestMagicLink(
  email: string,
  language?: string
): Promise<void> {
  await api.post("/api/auth/web/magic/request", { email, language })
}

export async function verifyMagicLink(token: string): Promise<string> {
  const { data } = await api.post<{ redirect: string }>(
    "/api/auth/web/magic/verify",
    { token }
  )
  return data.redirect || "/"
}

export async function logout(): Promise<void> {
  await api.post("/api/auth/web/logout")
}

export function ssoStartUrl(provider: "google" | "microsoft"): string {
  const base = import.meta.env.VITE_API_URL
  const prefix =
    typeof base === "string" && base.length > 0 ? base.replace(/\/$/, "") : ""
  return `${prefix}/api/auth/web/${provider}/start`
}
