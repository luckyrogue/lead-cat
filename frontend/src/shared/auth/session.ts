import type { TmaUser, TmaUserRole } from "./types"

const SESSION_KEY = "lc.tma.auth"

export type SessionData = {
  accessToken: string
  user: TmaUser
}

type TmaUserDto = {
  telegram_id: number
  name: string
  email: string
  role: TmaUserRole
}

function dtoToUser(dto: TmaUserDto): TmaUser {
  return {
    telegramId: dto.telegram_id,
    name: dto.name,
    email: dto.email,
    role: dto.role,
  }
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
    if (
      typeof parsed.accessToken !== "string" ||
      !parsed.user ||
      typeof parsed.user.email !== "string"
    ) {
      return null
    }
    return parsed
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

export function sessionFromAuthResponse(
  token: string,
  user: TmaUserDto
): SessionData {
  return { accessToken: token, user: dtoToUser(user) }
}
