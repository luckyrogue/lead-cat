import { api } from "~/shared/api/client"

export async function requestMagicLink(
  email: string,
  language?: string
): Promise<void> {
  await api.post("/api/auth/web/magic/request", { email, language })
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
