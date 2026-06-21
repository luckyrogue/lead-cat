import { useEffect, useRef } from "react"
import { Outlet } from "react-router"

import { AuthGate } from "~/components/tma-shell/auth-gate"
import { TabBar } from "~/components/tma-shell/tab-bar"
import { initTelegramViewport, syncTmaTheme } from "~/shared/tma/telegram-env"

export function TmaLayout() {
  const frameRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    initTelegramViewport()
    syncTmaTheme(frameRef.current)
  }, [])

  return (
    <div
      ref={frameRef}
      className="tma-frame mx-auto min-h-svh w-full max-w-md bg-background bg-tma-pattern"
    >
      <AuthGate>
        <main className="px-4 pt-5 pb-24">
          <Outlet />
        </main>
        <TabBar />
      </AuthGate>
    </div>
  )
}
