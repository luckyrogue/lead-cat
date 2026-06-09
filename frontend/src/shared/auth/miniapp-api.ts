import { apiFetch } from "@/shared/api/client"
import type { MiniAppAuthResponse } from "@/shared/api/types"
import {
  clearSession,
  getSession,
  sessionFromAuthResponse,
  setSession,
} from "@/shared/auth/session"
import type { MiniAppUser } from "@/shared/auth/types"
import { getTelegramWebApp } from "@/shared/miniapp/telegram-env"

export type { MiniAppUser } from "@/shared/auth/types"

const ME_TTL_MS = 60_000

let meCache: { user: MiniAppUser; expiresAt: number } | null = null

export async function fetchCurrentUser(signal?: AbortSignal): Promise<MiniAppUser> {
  if (meCache && meCache.expiresAt > Date.now()) {
    return meCache.user
  }

  const dto = await apiFetch<MiniAppAuthResponse["user"]>("/miniapp/me", { signal })
  const user: MiniAppUser = {
    telegramId: dto.telegram_id,
    name: dto.name,
    email: dto.email,
    role: dto.role,
  }
  meCache = { user, expiresAt: Date.now() + ME_TTL_MS }

  const session = getSession()
  if (session) {
    setSession({ ...session, user })
  }

  return user
}

export function invalidateMeCache() {
  meCache = null
}

export async function miniappLogin(initData: string): Promise<MiniAppUser> {
  const data = await apiFetch<MiniAppAuthResponse>("/auth/miniapp", {
    method: "POST",
    body: { init_data: initData },
  })
  const session = sessionFromAuthResponse(data.token, data.user)
  setSession(session)
  meCache = { user: session.user, expiresAt: Date.now() + ME_TTL_MS }
  return session.user
}

export function getInitData(): string {
  const fromTelegram = getTelegramWebApp()?.initData
  if (fromTelegram) return fromTelegram
  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return import.meta.env.VITE_MINIAPP_DEV_TG_ID ?? ""
  }
  return ""
}

export function logoutMiniAppSession() {
  clearSession()
  invalidateMeCache()
}
