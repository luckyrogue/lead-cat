import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { getInitData, miniappLogin } from "@/shared/auth/miniapp-api"
import type { MiniAppUser } from "@/shared/auth/types"
import { ApiError } from "@/shared/api/types"

export type MiniAppAuthStatus = "loading" | "authed" | "not_registered" | "error"

type MiniAppAuthValue = {
  status: MiniAppAuthStatus
  user: MiniAppUser | null
  errorCode: string | null
  retry: () => void
}

const MiniAppAuthContext = createContext<MiniAppAuthValue | null>(null)

export function MiniAppAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<MiniAppAuthStatus>("loading")
  const [user, setUser] = useState<MiniAppUser | null>(null)
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
    miniappLogin(initData)
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
    <MiniAppAuthContext.Provider value={{ status, user, errorCode, retry: run }}>
      {children}
    </MiniAppAuthContext.Provider>
  )
}

export function useMiniAppAuth(): MiniAppAuthValue {
  const ctx = useContext(MiniAppAuthContext)
  if (!ctx) throw new Error("useMiniAppAuth must be used within MiniAppAuthProvider")
  return ctx
}
