import { useState } from "react"

import { Button, CalendarPlus, toastError, toastSuccess } from "@leadcat/ui"

import { ListPageShell } from "~/components/list-page-shell"
import {
  useCreateMeeting,
  useDeleteMeeting,
  useUpdateMeeting,
} from "~/entities/meeting/mutations"
import { useMeetings } from "~/entities/meeting/queries"
import type {
  Meeting,
  MeetingListFilter,
  MeetingScope,
} from "~/entities/meeting/types"
import { useMembers } from "~/entities/org/queries"
import { MeetingCreateDialog } from "~/features/meetings/components/meeting-create-dialog"
import { MeetingDeleteDialog } from "~/features/meetings/components/meeting-delete-dialog"
import { MeetingEditDialog } from "~/features/meetings/components/meeting-edit-dialog"
import {
  MeetingsFilterBar,
  type OrganizerOption,
} from "~/features/meetings/components/meetings-filter-bar"
import { MeetingsTable } from "~/features/meetings/components/meetings-table"
import type { MeetingFormValues } from "~/features/meetings/components/meeting-form"
import {
  editDefaults,
  toCreateInput,
  toUpdateInput,
} from "~/features/meetings/pages/meetings-page-helpers"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { useDebouncedValue } from "~/shared/lib/use-debounced-value"
import { useT } from "~/shared/i18n/context"

export function MeetingsPage() {
  const t = useT()
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const createMeeting = useCreateMeeting(activeOrgId ?? "")
  const updateMeeting = useUpdateMeeting(activeOrgId ?? "")
  const deleteMeeting = useDeleteMeeting(activeOrgId ?? "")

  const [filter, setFilter] = useState<MeetingListFilter>({})
  const [deptInput, setDeptInput] = useState("")
  const debouncedDept = useDebouncedValue(deptInput, 300)
  const effectiveFilter: MeetingListFilter = {
    ...filter,
    dept: debouncedDept || undefined,
  }
  const members = useMembers(activeOrgId)
  const organizers: OrganizerOption[] = (members.data ?? [])
    .filter((member) => member.user_id)
    .map((member) => ({
      id: member.user_id as string,
      label: member.name || member.email,
    }))

  const meetings = useMeetings(activeOrgId, effectiveFilter)

  const [createOpen, setCreateOpen] = useState(false)
  const [toEdit, setToEdit] = useState<Meeting | null>(null)
  const [toDelete, setToDelete] = useState<Meeting | null>(null)
  const [deleteScope, setDeleteScope] = useState<MeetingScope>("this")

  function openDelete(meeting: Meeting) {
    setDeleteScope("this")
    setToDelete(meeting)
  }

  function handleCreate(values: MeetingFormValues) {
    createMeeting.mutate(toCreateInput(values), {
      onSuccess: () => {
        toastSuccess(t("meetings.toast.created"))
        setCreateOpen(false)
      },
      onError: (error) => toastError(error, t, "meetings.toast.createFailed"),
    })
  }

  function handleEdit(values: MeetingFormValues, scope: MeetingScope) {
    if (!toEdit) {
      return
    }
    updateMeeting.mutate(
      { meetingId: toEdit.id, values: toUpdateInput(values, scope), scope },
      {
        onSuccess: () => {
          toastSuccess(t("meetings.toast.updated"))
          setToEdit(null)
        },
        onError: (error) => toastError(error, t, "meetings.toast.updateFailed"),
      }
    )
  }

  function confirmDelete() {
    if (!toDelete) {
      return
    }
    deleteMeeting.mutate(
      { meetingId: toDelete.id, scope: deleteScope },
      {
        onSuccess: () => {
          toastSuccess(t("meetings.toast.cancelled"))
          setToDelete(null)
        },
        onError: (error) => {
          toastError(error, t, "meetings.toast.cancelFailed")
          setToDelete(null)
        },
      }
    )
  }

  return (
    <>
      <ListPageShell
        eyebrow={t("meetings.eyebrow")}
        title={t("meetings.title")}
        description={t("meetings.description")}
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <CalendarPlus className="size-4" />
            {t("meetings.newMeeting")}
          </Button>
        }
        isLoading={meetings.isPending}
        loadingMessage={t("meetings.loadingMeetings")}
        error={meetings.error}
        isEmpty={(meetings.data?.length ?? 0) === 0}
        emptyState={
          <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
            {t("meetings.emptyState")}
          </div>
        }
      >
        <MeetingsFilterBar
          filter={filter}
          dept={deptInput}
          organizers={organizers}
          onFilterChange={(patch) =>
            setFilter((current) => ({ ...current, ...patch }))
          }
          onDeptChange={setDeptInput}
        />
        <MeetingsTable
          meetings={meetings.data ?? []}
          pendingId={
            updateMeeting.isPending
              ? (updateMeeting.variables?.meetingId ?? null)
              : deleteMeeting.isPending
                ? (deleteMeeting.variables?.meetingId ?? null)
                : null
          }
          onEdit={setToEdit}
          onDelete={openDelete}
          timeZone={me?.user?.timezone || undefined}
        />
      </ListPageShell>

      <MeetingCreateDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        pending={createMeeting.isPending}
        onSubmit={handleCreate}
      />

      <MeetingEditDialog
        meeting={toEdit}
        orgId={activeOrgId}
        pending={updateMeeting.isPending}
        defaults={toEdit ? editDefaults(toEdit) : undefined}
        onOpenChange={(open) => !open && setToEdit(null)}
        onSubmit={handleEdit}
      />

      <MeetingDeleteDialog
        meeting={toDelete}
        scope={deleteScope}
        pending={deleteMeeting.isPending}
        onScopeChange={setDeleteScope}
        onOpenChange={(open) => !open && setToDelete(null)}
        onConfirm={confirmDelete}
      />
    </>
  )
}
