import {
  Button,
  CalendarDays,
  CalendarPlus,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Check,
  Users,
  type LucideIcon,
} from "@leadcat/ui"
import { useState } from "react"
import { Link } from "react-router"

import { useCalendarConnections } from "~/entities/calendar-connection/queries"
import { useMeetings } from "~/entities/meeting/queries"
import { useInvites, useMembers } from "~/entities/org/queries"
import { useT } from "~/shared/i18n/context"
import { allDone, computeSteps, doneCount, type ChecklistStepKey } from "../lib/steps"
import { dismissChecklist, isChecklistDismissed } from "../lib/dismissed"

const META: Record<ChecklistStepKey, { icon: LucideIcon; to: string }> = {
  calendar: { icon: CalendarDays, to: "/settings" },
  invite: { icon: Users, to: "/invites" },
  meeting: { icon: CalendarPlus, to: "/meetings" },
}

export function ActivationChecklist({
  activeOrgId,
}: {
  activeOrgId: string | null
}) {
  const t = useT()
  const [dismissed, setDismissed] = useState(isChecklistDismissed)
  const connections = useCalendarConnections()
  const members = useMembers(activeOrgId)
  const invites = useInvites(activeOrgId)
  const meetings = useMeetings(activeOrgId)

  if (!activeOrgId || dismissed) {
    return null
  }
  if (
    connections.isPending ||
    members.isPending ||
    invites.isPending ||
    meetings.isPending
  ) {
    return null
  }

  const steps = computeSteps({
    connections: connections.data ?? [],
    membersCount: members.data?.length ?? 0,
    invitesCount: invites.data?.length ?? 0,
    meetingsCount: meetings.data?.length ?? 0,
  })
  if (allDone(steps)) {
    return null
  }

  function onDismiss() {
    dismissChecklist()
    setDismissed(true)
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div>
          <CardTitle>{t("dashboard.checklist.title")}</CardTitle>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("dashboard.checklist.progress", {
              done: doneCount(steps),
              total: steps.length,
            })}
          </p>
        </div>
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          {t("dashboard.checklist.dismiss")}
        </Button>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {steps.map((step) => {
          const Icon = step.done ? Check : META[step.key].icon
          return (
            <div
              key={step.key}
              className="flex items-center justify-between gap-3"
            >
              <span className="flex items-center gap-2">
                <Icon
                  className={
                    step.done ? "size-4 text-primary" : "size-4 text-muted-foreground"
                  }
                />
                <span className={step.done ? "text-muted-foreground line-through" : ""}>
                  {t(`dashboard.checklist.${step.key}`)}
                </span>
              </span>
              {step.done ? null : (
                <Button asChild size="sm" variant="outline">
                  <Link to={META[step.key].to}>
                    {t(`dashboard.checklist.${step.key}Cta`)}
                  </Link>
                </Button>
              )}
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
