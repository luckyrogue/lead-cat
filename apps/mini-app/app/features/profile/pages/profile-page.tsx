import {
  Avatar,
  AvatarFallback,
  ArrowRight,
  Card,
  CardContent,
  Mail,
  Shield,
  Users,
} from "@leadcat/ui"
import { Link } from "react-router"

import { PageHeader } from "~/components/page-header"
import { PreferencesSettings } from "~/features/profile/components/preferences-settings"
import { ReminderSettings } from "~/features/profile/components/reminder-settings"
import { useAuth } from "~/shared/auth/auth-context"
import { useT } from "~/shared/i18n/context"

export function ProfilePage() {
  const t = useT()
  const { user } = useAuth()
  const initials = (user?.name ?? "?")
    .split(" ")
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join("")
    .toUpperCase()

  return (
    <div className="flex flex-col gap-5">
      <PageHeader title={t("profile.title")} />

      <Card>
        <CardContent className="flex items-center gap-3">
          <Avatar className="size-12">
            <AvatarFallback>{initials || "?"}</AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <p className="truncate font-semibold text-foreground">
              {user?.name || "—"}
            </p>
            <p className="flex items-center gap-1.5 truncate text-sm text-muted-foreground">
              <Mail className="size-3.5" />
              {user?.email || "—"}
            </p>
            {user?.role ? (
              <p className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                <Shield className="size-3" />
                {user.role}
              </p>
            ) : null}
          </div>
        </CardContent>
      </Card>

      <ReminderSettings />

      <PreferencesSettings />

      <Link to="/profile/colleague">
        <Card className="transition-colors active:bg-muted/50">
          <CardContent className="flex items-center justify-between gap-3">
            <span className="flex items-center gap-2 font-medium text-foreground">
              <Users className="size-4 text-muted-foreground" />
              {t("profile.colleagueSchedule")}
            </span>
            <ArrowRight className="size-4 text-muted-foreground" />
          </CardContent>
        </Card>
      </Link>
    </div>
  )
}
