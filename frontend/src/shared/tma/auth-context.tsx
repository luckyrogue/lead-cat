import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react"
import { isAxiosError } from "axios"
import { getInitData, tmaLogin, type TmaUser } from "./auth"

export type TmaAuthStatus = "loading" | "authed" | "not_registered" | "error"

type TmaAuthValue = {
  status: TmaAuthStatus
  user: TmaUser | null
  retry: () => void
}

const TmaAuthContext = createContext<TmaAuthValue | null>(null)

export function TmaAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<TmaAuthStatus>("loading")
  const [user, setUser] = useState<TmaUser | null>(null)

  function run() {
    setStatus("loading")
    const initData = getInitData()
    if (!initData) {
      setStatus("error")
      return
    }
    tmaLogin(initData)
      .then((u) => {
        setUser(u)
        setStatus("authed")
      })
      .catch((e) => {
        const code = isAxiosError(e)
          ? (e.response?.data as { code?: string })?.code
          : undefined
        setStatus(code === "not_registered" ? "not_registered" : "error")
      })
  }

  useEffect(() => {
    run()
    // run once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <TmaAuthContext.Provider value={{ status, user, retry: run }}>
      {children}
    </TmaAuthContext.Provider>
  )
}

export function useTmaAuth(): TmaAuthValue {
  const ctx = useContext(TmaAuthContext)
  if (!ctx) throw new Error("useTmaAuth must be used within TmaAuthProvider")
  return ctx
}
