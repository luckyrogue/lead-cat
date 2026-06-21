import { Button, Card, CardContent, cn } from "@leadcat/ui"
import { useEffect } from "react"

import {
  useCalendarConnections,
  useDisconnect,
  useStartConnect,
} from "~/entities/calendar-connection/queries"
import type { CalendarProvider } from "~/entities/calendar-connection/types"
import { useT } from "~/shared/i18n/context"
import { assertAllowedOAuthUrl } from "~/shared/lib/oauth-url"
import { toastError } from "~/shared/lib/toast"
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
  const microsoft = data.find((c) => c.provider === "microsoft")

  const connect = async (provider: CalendarProvider) => {
    try {
      const res = await start.mutateAsync(provider)
      const authUrl = assertAllowedOAuthUrl(res.auth_url)
      getWebApp()?.openLink?.(authUrl)
    } catch (error) {
      toastError(error, t, "profile.calendar.connectFailed")
    }
  }

  const disconnectProvider = (provider: CalendarProvider) => {
    disconnect.mutate(provider, {
      onError: (error) =>
        toastError(error, t, "profile.calendar.disconnectFailed"),
    })
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
                className={cn(
                  "shrink-0 text-destructive hover:text-destructive"
                )}
                onClick={() => disconnectProvider("google")}
              >
                {t("profile.calendar.disconnect")}
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              disabled={start.isPending}
              onClick={() => connect("google")}
            >
              {t("profile.calendar.connectGoogle")}
            </Button>
          )}
        </div>
        <div className="flex items-center justify-between gap-3">
          {microsoft?.connected ? (
            <>
              <span className="truncate text-sm text-muted-foreground">
                {t("profile.calendar.connected", { email: microsoft.email })}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={disconnect.isPending}
                className={cn(
                  "shrink-0 text-destructive hover:text-destructive"
                )}
                onClick={() => disconnectProvider("microsoft")}
              >
                {t("profile.calendar.disconnect")}
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              disabled={start.isPending}
              onClick={() => connect("microsoft")}
            >
              {t("profile.calendar.connectMicrosoft")}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
