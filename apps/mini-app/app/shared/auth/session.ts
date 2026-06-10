const SESSION_KEY = "leadcat.tma.session"

export type AuthUser = {
  telegram_id: number
  name: string
  email: string
  role: string
}

export type SessionData = {
  token: string
  user: AuthUser
}

export function getSession(): SessionData | null {
  if (typeof window === "undefined") {
    return null
  }
  const raw = sessionStorage.getItem(SESSION_KEY)
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as SessionData
    if (typeof parsed?.token === "string" && parsed.token.length > 0) {
      return parsed
    }
    return null
  } catch {
    return null
  }
}

export function setSession(data: SessionData): void {
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(data))
}

export function clearSession(): void {
  sessionStorage.removeItem(SESSION_KEY)
}
