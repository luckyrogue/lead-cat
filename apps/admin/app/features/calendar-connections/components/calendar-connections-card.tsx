import {
  Button,
  CalendarDays,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@leadcat/ui"

import {
  useCalendarConnections,
  useDisconnect,
  useStartConnect,
} from "~/entities/calendar-connection/queries"
import { useT } from "~/shared/i18n/context"

export function CalendarConnectionsCard() {
  const t = useT()
  const { data = [] } = useCalendarConnections()
  const start = useStartConnect()
  const disconnectMutation = useDisconnect()
  const google = data.find((c) => c.provider === "google")

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CalendarDays className="size-5 text-muted-foreground" />
          <CardTitle>{t("settings.calendars.title")}</CardTitle>
        </div>
        <CardDescription>{t("settings.calendars.subtitle")}</CardDescription>
      </CardHeader>
      <CardContent className="flex items-center gap-3">
        {google?.connected ? (
          <>
            <span className="text-sm text-muted-foreground">
              {t("settings.calendars.connected", { email: google.email })}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={disconnectMutation.isPending}
              onClick={() => disconnectMutation.mutate("google")}
            >
              {t("settings.calendars.disconnect")}
            </Button>
          </>
        ) : (
          <Button
            size="sm"
            disabled={start.isPending}
            onClick={() => start.mutate("google")}
          >
            {t("settings.calendars.connectGoogle")}
          </Button>
        )}
      </CardContent>
    </Card>
  )
}
