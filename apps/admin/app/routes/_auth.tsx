import { Navigate, Outlet } from "react-router"

import { BrandLogo } from "~/components/brand-logo"
import { PageLoading } from "~/components/page-loading"
import { useMe } from "~/shared/auth/use-me"

export default function AuthLayout() {
  const { data: me, isPending } = useMe()

  if (isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <PageLoading>Loading…</PageLoading>
      </div>
    )
  }

  if (me) {
    return <Navigate to="/" replace />
  }

  return (
    <div className="relative flex min-h-svh items-center justify-center overflow-hidden px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,oklch(0.9_0.07_55/0.45),transparent_28%),radial-gradient(circle_at_bottom_right,oklch(0.93_0.08_92/0.6),transparent_36%)]" />
      <div className="relative flex w-full max-w-md flex-col gap-6">
        <BrandLogo subtitle="Admin Panel" />
        <Outlet />
      </div>
    </div>
  )
}
