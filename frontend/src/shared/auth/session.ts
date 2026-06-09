import type { MiniAppUser, MiniAppUserRole } from "./types"

const SESSION_KEY = "lc.miniapp.auth"

export type SessionData = {
  accessToken: string
  user: MiniAppUser
}

type MiniAppUserDto = {
  telegram_id: number
  name: string
  email: string
  role: MiniAppUserRole
}

function dtoToUser(dto: MiniAppUserDto): MiniAppUser {
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
  user: MiniAppUserDto
): SessionData {
  return { accessToken: token, user: dtoToUser(user) }
}
