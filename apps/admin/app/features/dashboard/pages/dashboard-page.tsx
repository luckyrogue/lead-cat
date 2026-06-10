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

  const firstName = me?.user.email.split("@")[0] ?? "there"

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Lead Cat Admin"
        title={`Welcome, ${firstName}`}
        description={
          activeOrg
            ? `A calm snapshot of ${activeOrg.name}.`
            : "A calm snapshot of your workspace."
        }
      />

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title="Organization"
          value={activeOrg?.name ?? "—"}
          hint={activeOrg ? `Slug: ${activeOrg.slug}` : "No organization yet"}
          icon={Building2}
        />
        <StatCard
          title="Members"
          value={members.isPending ? "…" : (members.data?.length ?? 0)}
          hint="People with access to this organization"
          icon={Users}
        />
        <StatCard
          title="Pending invites"
          value={invites.isPending ? "…" : (invites.data?.length ?? 0)}
          hint="Invitations awaiting acceptance"
          icon={Mailbox}
        />
      </div>
    </div>
  )
}
