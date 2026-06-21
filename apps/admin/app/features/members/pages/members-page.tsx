import { useState } from "react"

import {
  Button,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@leadcat/ui"

import { ListPageShell } from "~/components/list-page-shell"
import {
  useMembers,
  useRemoveMember,
  useUpdateMemberRole,
} from "~/entities/org/queries"
import type { OrgMember, OrgRole } from "~/entities/org/types"
import { MembersTable } from "~/features/members/components/members-table"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useT } from "~/shared/i18n/context"
import { toastError, toastSuccess } from "~/shared/lib/toast"

export function MembersPage() {
  const t = useT()
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const members = useMembers(activeOrgId)
  const updateRole = useUpdateMemberRole(activeOrgId ?? "")
  const removeMember = useRemoveMember(activeOrgId ?? "")
  const [toRemove, setToRemove] = useState<OrgMember | null>(null)

  const currentRole = members.data?.find((m) => m.user_id === me?.user.id)?.role
  const canManage = currentRole === "owner" || currentRole === "admin"

  function handleRoleChange(userId: string, role: OrgRole) {
    updateRole.mutate(
      { userId, role },
      {
        onSuccess: () => toastSuccess(t("members.toast.roleUpdated")),
        onError: (error) =>
          toastError(error, t, "members.toast.roleUpdateFailed"),
      }
    )
  }

  function confirmRemove() {
    if (!toRemove || !toRemove.user_id) {
      return
    }
    removeMember.mutate(toRemove.user_id, {
      onSuccess: () => {
        toastSuccess(t("members.toast.removed"))
        setToRemove(null)
      },
      onError: (error) => {
        toastError(error, t, "members.toast.removeFailed")
        setToRemove(null)
      },
    })
  }

  return (
    <>
      <ListPageShell
        eyebrow={t("members.eyebrow")}
        title={t("members.title")}
        description={t("members.description")}
        isLoading={members.isPending}
        loadingMessage={t("members.loadingMembers")}
        error={members.error}
        isEmpty={(members.data?.length ?? 0) === 0}
        emptyState={
          <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
            {t("members.emptyState")}
          </div>
        }
      >
        <MembersTable
          members={members.data ?? []}
          canManage={canManage}
          pendingUserId={
            updateRole.isPending
              ? updateRole.variables?.userId
              : removeMember.isPending
                ? removeMember.variables
                : null
          }
          onRoleChange={handleRoleChange}
          onRemove={setToRemove}
        />
      </ListPageShell>

      <Dialog
        open={toRemove !== null}
        onOpenChange={(open) => !open && setToRemove(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("members.dialog.removeTitle")}</DialogTitle>
            <DialogDescription>
              {t("members.dialog.removeDescription")}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">{t("common.cancel")}</Button>
            </DialogClose>
            <Button
              variant="destructive"
              disabled={removeMember.isPending}
              onClick={confirmRemove}
            >
              {t("members.dialog.removeConfirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
