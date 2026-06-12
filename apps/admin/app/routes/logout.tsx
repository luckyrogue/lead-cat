import { useEffect } from "react"
import { useNavigate } from "react-router"

import { PageLoading } from "~/components/page-loading"
import { getQueryClient } from "~/shared/api/query-client"
import { setActiveOrgId } from "~/shared/api/active-org"
import { logout } from "~/shared/auth/api"
import { meQueryKey } from "~/shared/auth/use-me"

export default function LogoutPage() {
  const navigate = useNavigate()

  useEffect(() => {
    let active = true
    async function run() {
      await logout().catch(() => undefined)
      setActiveOrgId(null)
      getQueryClient().setQueryData(meQueryKey, null)
      getQueryClient().clear()
      if (active) {
        navigate("/login", { replace: true })
      }
    }
    void run()
    return () => {
      active = false
    }
  }, [navigate])

  return (
    <div className="flex min-h-svh items-center justify-center p-6">
      <PageLoading>Signing out…</PageLoading>
    </div>
  )
}
