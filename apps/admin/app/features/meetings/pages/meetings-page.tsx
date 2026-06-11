import { useState } from "react"

import {
  Button,
  CalendarPlus,
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Separator,
} from "@leadcat/ui"

import { ListPageShell } from "~/components/list-page-shell"
import {
  useCreateMeeting,
  useDeleteMeeting,
  useMeetings,
  useUpdateMeeting,
} from "~/entities/meeting/queries"
import {
  isSeries,
  type CreateMeetingInput,
  type Meeting,
  type MeetingScope,
  type UpdateMeetingInput,
} from "~/entities/meeting/types"
import {
  MeetingForm,
  type MeetingFormDefaults,
  type MeetingFormValues,
} from "~/features/meetings/components/meeting-form"
import { MeetingsTable } from "~/features/meetings/components/meetings-table"
import { ParticipantsEditor } from "~/features/meetings/components/participants-editor"
import { useActiveOrg } from "~/shared/auth/use-active-org"
import { useMe } from "~/shared/auth/use-me"
import { toastError, toastSuccess } from "~/shared/lib/toast"

function splitParticipants(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((email) => email.trim())
    .filter(Boolean)
}

function toCreateInput(values: MeetingFormValues): CreateMeetingInput {
  return {
    dept: values.dept,
    type: values.type,
    host: values.host || undefined,
    date: values.date,
    start: values.start,
    end: values.end,
    recurrence: values.recurrence,
    desc: values.desc || undefined,
    participants: splitParticipants(values.participants),
    recurrence_until:
      values.recurrence === "once" ? undefined : values.recurrence_until,
    recurrence_days:
      values.recurrence === "custom" ? values.recurrence_days : undefined,
  }
}

function toUpdateInput(
  values: MeetingFormValues,
  scope: MeetingScope
): UpdateMeetingInput {
  return {
    dept: values.dept,
    type: values.type,
    host: values.host || undefined,
    date: scope === "whole" ? undefined : values.date,
    start: values.start,
    end: values.end,
    desc: values.desc,
  }
}

function localDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ""
  }
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function localTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return ""
  }
  const pad = (n: number) => String(n).padStart(2, "0")
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function editDefaults(meeting: Meeting): MeetingFormDefaults {
  return {
    dept: meeting.dept,
    type: meeting.type,
    host: meeting.host,
    date: localDate(meeting.starts_at),
    start: localTime(meeting.starts_at),
    end: localTime(meeting.ends_at),
    desc: meeting.description,
  }
}

export function MeetingsPage() {
  const { data: me } = useMe()
  const { activeOrgId } = useActiveOrg(me?.organizations ?? [])
  const meetings = useMeetings(activeOrgId)
  const createMeeting = useCreateMeeting(activeOrgId ?? "")
  const updateMeeting = useUpdateMeeting(activeOrgId ?? "")
  const deleteMeeting = useDeleteMeeting(activeOrgId ?? "")

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
        toastSuccess("Meeting created.")
        setCreateOpen(false)
      },
      onError: (error) => toastError(error, "Could not create the meeting."),
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
          toastSuccess("Meeting updated.")
          setToEdit(null)
        },
        onError: (error) => toastError(error, "Could not update the meeting."),
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
          toastSuccess("Meeting cancelled.")
          setToDelete(null)
        },
        onError: (error) => {
          toastError(error, "Could not cancel the meeting.")
          setToDelete(null)
        },
      }
    )
  }

  return (
    <>
      <ListPageShell
        eyebrow="Organization"
        title="Meetings"
        description="Schedule and manage Google Meet sessions for your team."
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <CalendarPlus className="size-4" />
            New meeting
          </Button>
        }
        isLoading={meetings.isPending}
        loadingMessage="Loading meetings…"
        error={meetings.error}
        isEmpty={(meetings.data?.length ?? 0) === 0}
        emptyState={
          <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
            No meetings yet. Schedule one with the New meeting button.
          </div>
        }
      >
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
        />
      </ListPageShell>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>New meeting</DialogTitle>
            <DialogDescription>
              Schedule a Google Meet session and invite participants.
            </DialogDescription>
          </DialogHeader>
          <MeetingForm
            mode="create"
            pending={createMeeting.isPending}
            onSubmit={handleCreate}
          />
        </DialogContent>
      </Dialog>

      <Dialog
        open={toEdit !== null}
        onOpenChange={(open) => !open && setToEdit(null)}
      >
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Edit meeting</DialogTitle>
            <DialogDescription>
              Update the details of this meeting occurrence.
            </DialogDescription>
          </DialogHeader>
          {toEdit ? (
            <>
              <MeetingForm
                mode="edit"
                pending={updateMeeting.isPending}
                series={isSeries(toEdit)}
                defaults={editDefaults(toEdit)}
                onSubmit={handleEdit}
              />
              {activeOrgId ? (
                <>
                  <Separator />
                  <ParticipantsEditor
                    orgId={activeOrgId}
                    meetingId={toEdit.id}
                    series={isSeries(toEdit)}
                  />
                </>
              ) : null}
            </>
          ) : null}
        </DialogContent>
      </Dialog>

      <Dialog
        open={toDelete !== null}
        onOpenChange={(open) => !open && setToDelete(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cancel meeting?</DialogTitle>
            <DialogDescription>
              This cancels the meeting and removes it from connected calendars.
              This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          {toDelete && isSeries(toDelete) ? (
            <div className="space-y-2">
              <Label>Apply to</Label>
              <Select
                value={deleteScope}
                onValueChange={(value) => setDeleteScope(value as MeetingScope)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="this">This meeting only</SelectItem>
                  <SelectItem value="whole">The whole series</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ) : null}
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">Keep meeting</Button>
            </DialogClose>
            <Button
              variant="destructive"
              disabled={deleteMeeting.isPending}
              onClick={confirmDelete}
            >
              Cancel meeting
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
