import {
  Building2,
  Card,
  CardContent,
  Mailbox,
  Users,
  type LucideIcon,
} from "@leadcat/ui"

import { PageHeader } from "~/components/page-header"
import { useInvites, useMembers } from "~/entities/org/queries"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"
import { ActivationChecklist } from "../components/activation-checklist"

function StatCard({
  title,
  value,
  hint,
  icon: Icon,
}: {
  title: string
  value: string | number
  hint: string
  icon: LucideIcon
}) {
  return (
    <Card>
      <CardContent className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="mt-2 text-3xl font-semibold tracking-tight">{value}</p>
          <p className="mt-2 text-sm text-muted-foreground">{hint}</p>
        </div>
        <span className="rounded-[calc(var(--radius)*0.7)] border border-border/70 bg-background/70 p-2 text-primary">
          <Icon className="size-5" />
        </span>
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const { data: me } = useMe()
  const { activeOrg, activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const members = useMembers(activeOrgId)
  const invites = useInvites(activeOrgId)
  const t = useT()

  const firstName = me?.user.email.split("@")[0] ?? "there"

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow={t("dashboard.eyebrow")}
        title={t("dashboard.welcome", { name: firstName })}
        description={
          activeOrg
            ? t("dashboard.descriptionOrg", { org: activeOrg.name })
            : t("dashboard.descriptionDefault")
        }
      />

      <ActivationChecklist activeOrgId={activeOrgId} />

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title={t("dashboard.statOrganization")}
          value={activeOrg?.name ?? "—"}
          hint={
            activeOrg
              ? t("dashboard.statOrganizationHintSlug", {
                  slug: activeOrg.slug,
                })
              : t("dashboard.statOrganizationHintNone")
          }
          icon={Building2}
        />
        <StatCard
          title={t("dashboard.statMembers")}
          value={members.isPending ? "…" : (members.data?.length ?? 0)}
          hint={t("dashboard.statMembersHint")}
          icon={Users}
        />
        <StatCard
          title={t("dashboard.statInvites")}
          value={invites.isPending ? "…" : (invites.data?.length ?? 0)}
          hint={t("dashboard.statInvitesHint")}
          icon={Mailbox}
        />
      </div>
    </div>
  )
}
