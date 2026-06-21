import { Outlet } from "react-router"

import { AuthGate } from "~/components/tma-shell/auth-gate"
import { HtmlLangSync } from "~/components/tma-shell/html-lang-sync"
import { TabBar } from "~/components/tma-shell/tab-bar"

export function TmaLayout() {
  return (
    <div className="tma-frame mx-auto min-h-svh w-full max-w-md bg-background">
      <AuthGate>
        <HtmlLangSync />
        <main className="px-4 pt-5 pb-24">
          <Outlet />
        </main>
        <TabBar />
      </AuthGate>
    </div>
  )
}
