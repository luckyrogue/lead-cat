import { redirect } from "@tanstack/react-router"

import { fetchCurrentUser, getInitData, tmaLogin } from "@/features/auth/api"
import { getSession } from "@/features/auth/session"
import { ApiError } from "@/shared/api/types"

export type TmaAuthState = {
  user: Awaited<ReturnType<typeof requireTmaAuth>>["user"]
}

export async function requireTmaAuth(): Promise<{ user: NonNullable<ReturnType<typeof getSession>>["user"] }> {
  const session = getSession()
  if (session?.accessToken) {
    return { user: session.user }
  }

  const initData = getInitData()
  if (!initData) {
    throw redirect({ to: "/" })
  }

  try {
    const user = await tmaLogin(initData)
    return { user }
  } catch (error) {
    if (error instanceof ApiError && error.code === "not_registered") {
      throw redirect({ to: "/" })
    }
    throw error
  }
}

export async function ensureTmaUser() {
  const session = getSession()
  if (session?.user) {
    return session.user
  }
  return fetchCurrentUser()
}
