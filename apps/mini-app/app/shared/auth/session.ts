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

type StoredSession = {
  user: AuthUser
}

let memoryToken: string | null = null

export function getToken(): string | null {
  return memoryToken
}

export function getStoredUser(): AuthUser | null {
  if (typeof window === "undefined") {
    return null
  }
  const raw = sessionStorage.getItem(SESSION_KEY)
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as StoredSession
    if (parsed?.user && typeof parsed.user.telegram_id === "number") {
      return parsed.user
    }
    return null
  } catch {
    return null
  }
}

export function getSession(): SessionData | null {
  const token = getToken()
  const user = getStoredUser()
  if (token && user) {
    return { token, user }
  }
  return null
}

export function setSession(data: SessionData): void {
  memoryToken = data.token
  if (typeof window !== "undefined") {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify({ user: data.user }))
  }
}

export function clearSession(): void {
  memoryToken = null
  if (typeof window !== "undefined") {
    sessionStorage.removeItem(SESSION_KEY)
  }
}
