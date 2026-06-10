import { useEffect } from "react"
import { Outlet, useNavigate } from "@tanstack/react-router"
import { useWebAuth } from "@/shared/web-auth/context"
import { Spinner } from "@/components/ui/spinner"
import { OrgSwitcher } from "./org-switcher"

export function WebLayout() {
  const { status, user, organizations, logout } = useWebAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (status === "anonymous") {
      void navigate({ to: "/login" })
    } else if (status === "authed" && organizations.length === 0) {
      void navigate({ to: "/onboarding" })
    }
    // status === "authed" && organizations.length > 0 → stay in layout, render Outlet
  }, [status, organizations.length, navigate])

  if (status === "loading") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Spinner className="size-6 text-gray-400" />
      </div>
    )
  }

  // While redirecting (anonymous / no orgs) render nothing to avoid flash.
  if (status === "anonymous" || (status === "authed" && organizations.length === 0)) {
    return null
  }

  function handleLogout() {
    logout()
    void navigate({ to: "/login" })
  }

  return (
    <div className="flex min-h-screen flex-col bg-gray-50">
      <header className="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-3">
        <div className="flex items-center gap-4">
          <span className="text-sm font-bold text-gray-900">Lead Cat</span>
          <OrgSwitcher />
        </div>
        <div className="flex items-center gap-4">
          {user && (
            <span className="text-sm text-gray-500">{user.email}</span>
          )}
          <button
            type="button"
            onClick={handleLogout}
            className="rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 active:bg-gray-100"
          >
            Log out
          </button>
        </div>
      </header>
      <main className="flex-1 p-6">
        <Outlet />
      </main>
    </div>
  )
}
