import { createFileRoute } from "@tanstack/react-router"
import { useWebAuth } from "@/shared/web-auth/context"

function DashboardPage() {
  const { user, organizations, activeOrgId } = useWebAuth()
  const activeOrg = organizations.find((o) => o.id === activeOrgId)

  return (
    <div className="space-y-2">
      <h2 className="text-xl font-semibold text-gray-900">Dashboard</h2>
      <p className="text-sm text-gray-500">
        Welcome, <strong>{user?.email}</strong>
        {activeOrg ? ` — ${activeOrg.name}` : ""}
      </p>
    </div>
  )
}

export const Route = createFileRoute("/_web/dashboard")({
  component: DashboardPage,
})
