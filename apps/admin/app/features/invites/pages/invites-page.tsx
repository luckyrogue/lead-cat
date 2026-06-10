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
import { toastError, toastSuccess } from "~/shared/lib/toast"

export function InvitesPage() {
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const invites = useInvites(activeOrgId)
  const createInvite = useCreateInvite(activeOrgId ?? "")
  const deleteInvite = useDeleteInvite(activeOrgId ?? "")

  function handleCreate(values: { email: string; role: OrgRole }) {
    createInvite.mutate(values, {
      onSuccess: () => toastSuccess(`Invited ${values.email}.`),
      onError: (error) => toastError(error, "Could not send the invite."),
    })
  }

  function handleDelete(invite: OrgInvite) {
    deleteInvite.mutate(invite.id, {
      onSuccess: () => toastSuccess("Invite revoked."),
      onError: (error) => toastError(error, "Could not revoke the invite."),
    })
  }

  const list = invites.data ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Organization"
        title="Invites"
        description="Invite teammates by email and manage pending invitations."
      />

      <Card>
        <CardHeader>
          <CardTitle>Invite a teammate</CardTitle>
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
          <CardTitle>Pending invites</CardTitle>
        </CardHeader>
        <CardContent>
          {invites.isPending ? (
            <PageLoading>Loading invites…</PageLoading>
          ) : invites.error ? (
            <p
              className="rounded-[calc(var(--radius)*1.15)] border border-destructive/20 bg-destructive/5 px-4 py-5 text-sm text-destructive"
              role="alert"
            >
              {invites.error instanceof Error
                ? invites.error.message
                : "Failed to load invites."}
            </p>
          ) : list.length === 0 ? (
            <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
              No pending invites.
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
