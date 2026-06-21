import { Suspense, useEffect } from "react"
import { Navigate, Outlet, useLocation } from "react-router"

import { AppSidebar } from "~/components/app-sidebar"
import { PageLoading } from "~/components/page-loading"
import { ensureActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { LocaleGate } from "~/shared/i18n/locale-gate"
import { useT } from "~/shared/i18n/context"

export default function AppLayout() {
  return (
    <LocaleGate>
      <AppLayoutBody />
    </LocaleGate>
  )
}

function AppLayoutBody() {
  const t = useT()
  const location = useLocation()
  const { data: me, isPending } = useMe()

  useEffect(() => {
    if (me) {
      ensureActiveOrg(me.organizations)
    }
  }, [me])

  if (isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <PageLoading>{t("sidebar.loadingWorkspace")}</PageLoading>
      </div>
    )
  }

  if (!me) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  if (me.organizations.length === 0) {
    return <Navigate to="/onboarding" replace />
  }

  return (
    <div className="relative min-h-svh xl:h-svh">
      <div className="mx-auto w-full max-w-[88rem] px-(--page-padding-inline,1.5rem) py-4 sm:py-5 xl:h-full">
        <div className="grid min-h-[calc(100svh-2rem)] gap-4 xl:h-[calc(100svh-2.5rem)] xl:min-h-0 xl:grid-cols-[18rem_minmax(0,1fr)]">
          <AppSidebar me={me} />
          <main className="min-w-0 xl:h-full xl:min-h-0">
            <div className="surface-card min-h-full rounded-[calc(var(--radius)*1.75)] px-4 py-4 sm:px-6 sm:py-6 lg:px-8 lg:py-8">
              <Suspense
                fallback={<PageLoading>{t("common.loading")}</PageLoading>}
              >
                <Outlet />
              </Suspense>
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}
