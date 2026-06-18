import { Button, Card, CardContent, cn } from "@leadcat/ui"
import { useEffect } from "react"

import { useCalendarConnections, useDisconnect, useStartConnect } from "~/entities/calendar-connection/queries"
import { useT } from "~/shared/i18n/context"
import { getWebApp } from "~/shared/tma/telegram-env"

export function CalendarConnectionRow() {
  const t = useT()
  const { data = [], refetch } = useCalendarConnections()
  const start = useStartConnect()
  const disconnect = useDisconnect()

  useEffect(() => {
    const onFocus = () => refetch()
    window.addEventListener("focus", onFocus)
    return () => window.removeEventListener("focus", onFocus)
  }, [refetch])

  const google = data.find((c) => c.provider === "google")

  const connect = async () => {
    const res = await start.mutateAsync("google")
    getWebApp()?.openLink?.(res.auth_url)
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm font-semibold text-foreground">
          {t("profile.calendar.title")}
        </p>
        <div className="flex items-center justify-between gap-3">
          {google?.connected ? (
            <>
              <span className="truncate text-sm text-muted-foreground">
                {t("profile.calendar.connected", { email: google.email })}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={disconnect.isPending}
                className={cn("shrink-0 text-destructive hover:text-destructive")}
                onClick={() => disconnect.mutate("google")}
              >
                {t("profile.calendar.disconnect")}
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              disabled={start.isPending}
              onClick={connect}
            >
              {t("profile.calendar.connectGoogle")}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
