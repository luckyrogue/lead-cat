import { getInitData, miniappLogin } from "@/shared/auth/miniapp-api"
import { getSession } from "@/shared/auth/session"
import { ApiError } from "@/shared/api/types"

let inflightRefresh: Promise<string | null> | null = null

async function refreshAccessToken(): Promise<string | null> {
  const initData = getInitData()
  if (!initData) {
    return null
  }

  try {
    await miniappLogin(initData)
    return getSession()?.accessToken ?? null
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      return null
    }
    throw error
  }
}

export async function refreshMiniAppSessionIfNeeded({
  force = false,
}: { force?: boolean } = {}): Promise<string | null> {
  const current = getSession()?.accessToken
  if (!force && current) {
    return current
  }

  if (inflightRefresh) {
    return inflightRefresh
  }

  inflightRefresh = refreshAccessToken().finally(() => {
    inflightRefresh = null
  })

  return inflightRefresh
}
