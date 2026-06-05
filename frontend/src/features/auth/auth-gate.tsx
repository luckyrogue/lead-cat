import { useNavigate, useRouterState } from "@tanstack/react-router"
import { useEffect } from "react"
import { setAuthToken } from "@/shared/api/client"
import { getAccessToken, isAuthenticated } from "@/shared/auth/session"

type Props = {
  children: React.ReactNode
}

export function AuthGate({ children }: Props) {
  const navigate = useNavigate()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  const isTma = pathname === "/"

  useEffect(() => {
    const token = getAccessToken()
    if (token) {
      setAuthToken(token)
    }
    if (pathname === "/login") {
      if (isAuthenticated()) {
        navigate({ to: "/workspaces" })
      }
      return
    }
    if (isTma) return
    if (!isAuthenticated()) {
      navigate({ to: "/login" })
    }
  }, [pathname, navigate, isTma])

  if (pathname === "/login") {
    return <>{children}</>
  }
  if (isTma) {
    return <>{children}</>
  }
  if (!isAuthenticated()) {
    return null
  }
  return <>{children}</>
}
