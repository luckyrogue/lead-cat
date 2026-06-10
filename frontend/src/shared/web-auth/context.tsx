import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { useMe, useLogout } from "./queries"
import { ACTIVE_ORG_KEY } from "./api"
import type { WebOrg, WebUser } from "./api"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type WebAuthStatus = "loading" | "authed" | "anonymous" | "error"

type WebAuthValue = {
  status: WebAuthStatus
  user: WebUser | null
  organizations: WebOrg[]
  activeOrgId: string | null
  setActiveOrgId: (id: string) => void
  logout: () => void
}

// ---------------------------------------------------------------------------
// localStorage helpers (key imported from api.ts — single source of truth)
// ---------------------------------------------------------------------------

function loadStoredOrgId(): string | null {
  try {
    return localStorage.getItem(ACTIVE_ORG_KEY) ?? null
  } catch {
    return null
  }
}

function saveOrgId(id: string): void {
  try {
    localStorage.setItem(ACTIVE_ORG_KEY, id)
  } catch {
    // ignore
  }
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

const WebAuthContext = createContext<WebAuthValue | null>(null)

export function WebAuthProvider({ children }: { children: ReactNode }) {
  const { data, isLoading, isError, error } = useMe()
  const logoutMutation = useLogout()

  const [activeOrgId, setActiveOrgIdState] = useState<string | null>(
    loadStoredOrgId
  )

  // Derive status from query state. 401 → anonymous; other errors → error.
  let status: WebAuthStatus = "loading"
  if (isLoading) {
    status = "loading"
  } else if (isError) {
    // axios wraps HTTP errors; check for 401
    const httpStatus =
      (error as { status?: number; response?: { status?: number } })?.status ??
      (error as { response?: { status?: number } })?.response?.status ??
      0
    status = httpStatus === 401 ? "anonymous" : "error"
  } else if (data) {
    status = "authed"
  }

  const organizations = data?.organizations ?? []
  const user = data?.user ?? null

  // When authed and no stored org, default to the first org.
  useEffect(() => {
    if (status === "authed" && organizations.length > 0 && !loadStoredOrgId()) {
      const firstId = organizations[0].id
      setActiveOrgIdState(firstId)
      saveOrgId(firstId)
    }
  }, [status, organizations])

  const setActiveOrgId = useCallback((id: string) => {
    setActiveOrgIdState(id)
    saveOrgId(id)
  }, [])

  const handleLogout = useCallback(() => {
    logoutMutation.mutate()
  }, [logoutMutation])

  return (
    <WebAuthContext.Provider
      value={{
        status,
        user,
        organizations,
        activeOrgId,
        setActiveOrgId,
        logout: handleLogout,
      }}
    >
      {children}
    </WebAuthContext.Provider>
  )
}

export function useWebAuth(): WebAuthValue {
  const ctx = useContext(WebAuthContext)
  if (!ctx) throw new Error("useWebAuth must be used within WebAuthProvider")
  return ctx
}
