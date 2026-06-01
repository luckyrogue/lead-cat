import { isAxiosError } from "axios"
import { api, setAuthToken } from "@/shared/api/client"

export type TmaUser = {
  telegramId: number
  name: string
  email: string
  role: "user" | "admin"
}

type TmaAuthResponse = {
  token: string
  user: {
    telegram_id: number
    name: string
    email: string
    role: "user" | "admin"
  }
}

// tmaLogin exchanges Telegram initData for a TMA JWT and sets it as the axios
// bearer token. Returns the authenticated user.
export async function tmaLogin(initData: string): Promise<TmaUser> {
  const res = await api.post<TmaAuthResponse>("/auth/tma", {
    init_data: initData,
  })
  const { token, user } = res.data
  setAuthToken(token)
  return {
    telegramId: user.telegram_id,
    name: user.name,
    email: user.email,
    role: user.role,
  }
}

// getInitData returns the Telegram WebApp initData string, or — in dev mode —
// the configured dev telegram id (the backend dev path treats init_data as the id).
export function getInitData(): string {
  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return import.meta.env.VITE_TMA_DEV_TG_ID ?? ""
  }
  const tg = (
    window as unknown as { Telegram?: { WebApp?: { initData?: string } } }
  ).Telegram
  return tg?.WebApp?.initData ?? ""
}

let interceptorInstalled = false

// installTmaAuthInterceptor wires a one-shot re-login on 401 from /api/tma/* calls:
// it re-exchanges initData for a fresh TMA JWT once and replays the request.
export function installTmaAuthInterceptor(): void {
  if (interceptorInstalled) return
  interceptorInstalled = true
  api.interceptors.response.use(undefined, async (error: unknown) => {
    if (!isAxiosError(error) || !error.config) return Promise.reject(error)
    const cfg = error.config as typeof error.config & { __tmaRetried?: boolean }
    const url = cfg.url ?? ""
    if (
      error.response?.status === 401 &&
      url.startsWith("/tma/") &&
      !cfg.__tmaRetried
    ) {
      cfg.__tmaRetried = true
      try {
        await tmaLogin(getInitData())
        return api(cfg)
      } catch {
        // Re-login failed; surface a non-PII signal (never log the error object —
        // it carries init_data) and fall through to the original 401 rejection.
        console.warn("tma re-login failed")
      }
    }
    return Promise.reject(error)
  })
}
