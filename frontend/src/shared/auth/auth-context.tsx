import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { getInitData, tmaLogin } from "@/shared/auth/tma-api"
import type { TmaUser } from "@/shared/auth/types"
import { ApiError } from "@/shared/api/types"

export type TmaAuthStatus = "loading" | "authed" | "not_registered" | "error"

type TmaAuthValue = {
  status: TmaAuthStatus
  user: TmaUser | null
  errorCode: string | null
  retry: () => void
}

const TmaAuthContext = createContext<TmaAuthValue | null>(null)

export function TmaAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<TmaAuthStatus>("loading")
  const [user, setUser] = useState<TmaUser | null>(null)
  const [errorCode, setErrorCode] = useState<string | null>(null)

  function run() {
    setStatus("loading")
    setErrorCode(null)
    const initData = getInitData()
    if (!initData) {
      setStatus("error")
      setErrorCode("no_init_data")
      return
    }
    tmaLogin(initData)
      .then((u) => {
        setUser(u)
        setStatus("authed")
      })
      .catch((e) => {
        const code = e instanceof ApiError ? e.code : undefined
        setErrorCode(code ?? "unknown")
        setStatus(code === "not_registered" ? "not_registered" : "error")
      })
  }

  useEffect(() => {
    run()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <TmaAuthContext.Provider value={{ status, user, errorCode, retry: run }}>
      {children}
    </TmaAuthContext.Provider>
  )
}

export function useTmaAuth(): TmaAuthValue {
  const ctx = useContext(TmaAuthContext)
  if (!ctx) throw new Error("useTmaAuth must be used within TmaAuthProvider")
  return ctx
}
