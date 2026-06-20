import { useState } from "react"
import {
  Badge,
  Button,
  CalendarClock,
  Check,
  Link2,
  Loader2,
  Pencil,
  Trash2,
} from "@leadcat/ui"

import {
  useMyEventTypes,
  useCreateEventType,
  useUpdateEventType,
  useDeleteEventType,
} from "~/entities/booking-event-type/queries"
import type {
  BookingEventType,
  EventTypeInput,
} from "~/entities/booking-event-type/types"
import { EventTypeDialog } from "~/features/booking/components/event-type-dialog"
import { ListPageShell } from "~/components/list-page-shell"
import { toastError, toastSuccess } from "~/shared/lib/toast"
import { useT } from "~/shared/i18n/context"

export function BookingPage() {
  const t = useT()
  const { data: eventTypes, isPending, error } = useMyEventTypes()
  const createEventType = useCreateEventType()
  const updateEventType = useUpdateEventType()
  const deleteEventType = useDeleteEventType()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<BookingEventType | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)

  function openCreate() {
    setEditing(null)
    setDialogOpen(true)
  }

  function openEdit(et: BookingEventType) {
    setEditing(et)
    setDialogOpen(true)
  }

  function handleSubmit(input: EventTypeInput) {
    if (editing) {
      updateEventType.mutate(
        { id: editing.id, input },
        {
          onSuccess: () => {
            toastSuccess(t("booking.toast.updated"))
            setDialogOpen(false)
            setEditing(null)
          },
          onError: (err) => toastError(err, t("booking.toast.updateFailed")),
        }
      )
    } else {
      createEventType.mutate(input, {
        onSuccess: () => {
          toastSuccess(t("booking.toast.created"))
          setDialogOpen(false)
        },
        onError: (err) => toastError(err, t("booking.toast.createFailed")),
      })
    }
  }

  function handleDelete(et: BookingEventType) {
    deleteEventType.mutate(et.id, {
      onSuccess: () => toastSuccess(t("booking.toast.deleted")),
      onError: (err) => toastError(err, t("booking.toast.deleteFailed")),
    })
  }

  function handleCopyLink(et: BookingEventType) {
    const link = `${window.location.origin}/book/${et.slug}`
    navigator.clipboard.writeText(link).then(() => {
      setCopiedId(et.id)
      setTimeout(() => setCopiedId(null), 2000)
    })
  }

  return (
    <>
      <ListPageShell
        eyebrow={t("booking.eyebrow")}
        title={t("booking.title")}
        description={t("booking.description")}
        actions={
          <Button onClick={openCreate}>
            <CalendarClock className="size-4" />
            {t("booking.newEventType")}
          </Button>
        }
        isLoading={isPending}
        loadingMessage={t("booking.loading")}
        error={error}
        isEmpty={(eventTypes?.length ?? 0) === 0}
        emptyState={
          <div className="rounded-[calc(var(--radius)*1.15)] border border-dashed border-border/80 bg-muted/30 px-4 py-8 text-center text-sm text-muted-foreground">
            {t("booking.empty")}
          </div>
        }
      >
        <div className="flex flex-col gap-3">
          {(eventTypes ?? []).map((et) => {
            const link = `${window.location.origin}/book/${et.slug}`
            const isCopied = copiedId === et.id
            const isPendingDelete =
              deleteEventType.isPending && deleteEventType.variables === et.id

            return (
              <div
                key={et.id}
                className="flex flex-col gap-3 rounded-[calc(var(--radius)*1.15)] border border-border bg-background p-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-foreground">
                      {et.title}
                    </span>
                    <Badge tone={et.active ? "sunny" : "muted"}>
                      {et.active ? t("booking.active") : t("booking.inactive")}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {et.duration_mins} {t("booking.fields.minutes")}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Link2 className="size-3 text-muted-foreground" />
                    <a
                      href={link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="truncate text-xs text-muted-foreground hover:text-foreground hover:underline"
                    >
                      {link}
                    </a>
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleCopyLink(et)}
                  >
                    {isCopied ? (
                      <Check className="size-4" />
                    ) : (
                      <Link2 className="size-4" />
                    )}
                    {isCopied ? t("booking.linkCopied") : t("booking.copyLink")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => openEdit(et)}
                  >
                    <Pencil className="size-4" />
                    {t("common.edit")}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={isPendingDelete}
                    onClick={() => handleDelete(et)}
                  >
                    {isPendingDelete ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Trash2 className="size-4" />
                    )}
                    {t("common.delete")}
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      </ListPageShell>

      <EventTypeDialog
        open={dialogOpen}
        onOpenChange={(next) => {
          setDialogOpen(next)
          if (!next) setEditing(null)
        }}
        pending={createEventType.isPending || updateEventType.isPending}
        editing={editing}
        onSubmit={handleSubmit}
      />
    </>
  )
}
