import { apiFetch } from "@/shared/api/client"
import type { TmaAuthResponse } from "@/shared/api/types"

import {
  clearSession,
  getSession,
  sessionFromAuthResponse,
  setSession,
} from "@/shared/auth/session"
import type { TmaUser } from "@/shared/auth/types"

export type { TmaUser } from "@/shared/auth/types"

const ME_TTL_MS = 60_000

let meCache: { user: TmaUser; expiresAt: number } | null = null

export async function fetchCurrentUser(signal?: AbortSignal): Promise<TmaUser> {
  if (meCache && meCache.expiresAt > Date.now()) {
    return meCache.user
  }

  const dto = await apiFetch<TmaAuthResponse["user"]>("/tma/me", { signal })
  const user: TmaUser = {
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

export async function tmaLogin(initData: string): Promise<TmaUser> {
  const data = await apiFetch<TmaAuthResponse>("/auth/tma", {
    method: "POST",
    body: { init_data: initData },
  })
  const session = sessionFromAuthResponse(data.token, data.user)
  setSession(session)
  meCache = { user: session.user, expiresAt: Date.now() + ME_TTL_MS }
  return session.user
}

export function getInitData(): string {
  const fromTelegram = (
    window as unknown as { Telegram?: { WebApp?: { initData?: string } } }
  ).Telegram?.WebApp?.initData
  if (fromTelegram) return fromTelegram
  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return import.meta.env.VITE_TMA_DEV_TG_ID ?? ""
  }
  return ""
}

export function logoutTmaSession() {
  clearSession()
  invalidateMeCache()
}
