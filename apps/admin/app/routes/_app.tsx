import { Outlet } from "react-router"

import { AppSidebar } from "~/components/app-sidebar"

export default function AppLayout() {
  return (
    <div className="relative min-h-svh xl:h-svh">
      <div className="mx-auto w-full max-w-[88rem] px-(--page-padding-inline,1.5rem) py-4 sm:py-5 xl:h-full">
        <div className="grid min-h-[calc(100svh-2rem)] gap-4 xl:h-[calc(100svh-2.5rem)] xl:min-h-0 xl:grid-cols-[18rem_minmax(0,1fr)]">
          <AppSidebar />
          <main className="min-w-0 xl:h-full xl:min-h-0">
            <div className="surface-card min-h-full rounded-[calc(var(--radius)*1.75)] px-4 py-4 sm:px-6 sm:py-6 lg:px-8 lg:py-8">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </div>
  )
}
