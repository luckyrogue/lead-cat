import { redirect } from "@tanstack/react-router"

import { fetchCurrentUser, getInitData, miniappLogin } from "@/shared/auth/miniapp-api"
import { getSession } from "@/shared/auth/session"
import { ApiError } from "@/shared/api/types"

export type TmaAuthState = {
  user: Awaited<ReturnType<typeof requireMiniAppAuth>>["user"]
}

export async function requireMiniAppAuth(): Promise<{
  user: NonNullable<ReturnType<typeof getSession>>["user"]
}> {
  const session = getSession()
  if (session?.accessToken) {
    return { user: session.user }
  }

  const initData = getInitData()
  if (!initData) {
    throw redirect({ to: "/" })
  }

  try {
    const user = await miniappLogin(initData)
    return { user }
  } catch (error) {
    if (error instanceof ApiError && error.code === "not_registered") {
      throw redirect({ to: "/" })
    }
    throw error
  }
}

export async function ensureMiniAppUser() {
  const session = getSession()
  if (session?.user) {
    return session.user
  }
  return fetchCurrentUser()
}
