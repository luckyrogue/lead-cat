import { Card, CardContent, CardHeader, CardTitle } from "@leadcat/ui"

import { PageHeader } from "~/components/page-header"
import { PageLoading } from "~/components/page-loading"
import {
  useCreateInvite,
  useDeleteInvite,
  useInvites,
} from "~/entities/org/queries"
import type { OrgInvite, OrgRole } from "~/entities/org/types"
import { InviteForm } from "~/features/invites/components/invite-form"
import { InvitesTable } from "~/features/invites/components/invites-table"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"
import { toastError, toastSuccess } from "~/shared/lib/toast"

export function InvitesPage() {
  const t = useT()
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const invites = useInvites(activeOrgId)
  const createInvite = useCreateInvite(activeOrgId ?? "")
  const deleteInvite = useDeleteInvite(activeOrgId ?? "")

  function handleCreate(values: { email: string; role: OrgRole }) {
    createInvite.mutate(values, {
      onSuccess: () =>
        toastSuccess(t("invites.toast.invited", { email: values.email })),
      onError: (error) => toastError(error, t("invites.toast.inviteFailed")),
    })
  }

  function handleDelete(invite: OrgInvite) {
    deleteInvite.mutate(invite.id, {
      onSuccess: () => toastSuccess(t("invites.toast.revoked")),
      onError: (error) => toastError(error, t("invites.toast.revokeFailed")),
    })
  }

  const list = invites.data ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow={t("invites.eyebrow")}
        title={t("invites.title")}
        description={t("invites.description")}
      />

      <Card>
        <CardHeader>
          <CardTitle>{t("invites.cardInviteTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          <InviteForm
            pending={createInvite.isPending}
            onSubmit={handleCreate}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("invites.cardPendingTitle")}</CardTitle>
        </CardHeader>
        <CardContent>
          {invites.isPending ? (
            <PageLoading>{t("invites.loadingInvites")}</PageLoading>
          ) : invites.error ? (
            <p
              className="rounded-[calc(var(--radius)*1.15)] border border-destructive/20 bg-destructive/5 px-4 py-5 text-sm text-destructive"
              role="alert"
            >
              {invites.error instanceof Error
                ? invites.error.message
                : t("invites.failedToLoad")}
            </p>
          ) : list.length === 0 ? (
            <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
              {t("invites.emptyState")}
            </div>
          ) : (
            <InvitesTable
              invites={list}
              pendingId={deleteInvite.isPending ? deleteInvite.variables : null}
              onDelete={handleDelete}
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
