import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"

import { setReauthHandler } from "~/shared/api/client"
import { ApiError } from "~/shared/api/types"
import { authenticate } from "~/shared/auth/tma-api"
import {
  clearSession,
  getSession,
  setSession,
  type AuthUser,
} from "~/shared/auth/session"
import { getInitData } from "~/shared/tma/telegram-env"

export type AuthStatus = "loading" | "authed" | "not_registered" | "error"

type AuthState = {
  status: AuthStatus
  user: AuthUser | null
  error: string | null
  retry: () => void
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>("loading")
  const [user, setUser] = useState<AuthUser | null>(null)
  const [error, setError] = useState<string | null>(null)
  const inFlight = useRef<Promise<string | null> | null>(null)

  const runAuth = useCallback(async (): Promise<string | null> => {
    const initData = getInitData()
    if (!initData) {
      setStatus("error")
      setError("missing_init_data")
      return null
    }
    try {
      const session = await authenticate(initData)
      setSession(session)
      setUser(session.user)
      setStatus("authed")
      setError(null)
      return session.token
    } catch (err) {
      clearSession()
      setUser(null)
      if (err instanceof ApiError && err.code === "not_registered") {
        setStatus("not_registered")
        setError(null)
      } else {
        setStatus("error")
        setError(err instanceof ApiError ? err.message : "auth_failed")
      }
      return null
    }
  }, [])

  const authenticateOnce = useCallback((): Promise<string | null> => {
    if (!inFlight.current) {
      inFlight.current = runAuth().finally(() => {
        inFlight.current = null
      })
    }
    return inFlight.current
  }, [runAuth])

  const retry = useCallback(() => {
    setStatus("loading")
    setError(null)
    void authenticateOnce()
  }, [authenticateOnce])

  useEffect(() => {
    setReauthHandler(authenticateOnce)
    const existing = getSession()
    if (existing) {
      setUser(existing.user)
      setStatus("authed")
    } else {
      void authenticateOnce()
    }
    return () => setReauthHandler(null)
  }, [authenticateOnce])

  const value = useMemo<AuthState>(
    () => ({ status, user, error, retry }),
    [status, user, error, retry]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider")
  }
  return ctx
}
