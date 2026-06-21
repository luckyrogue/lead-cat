import {
  Button,
  DatePicker,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  MeetingWhenPicker,
  toast,
} from "@leadcat/ui"
import { useEffect, useState } from "react"

import {
  useChangeSeriesEnd,
  useUpdateMeeting,
} from "~/entities/meeting/mutations"
import {
  isSeriesMeeting,
  type Meeting,
  type MeetingMutationScope,
} from "~/entities/meeting/types"
import { ParticipantsEditor } from "~/features/meetings/components/participants-editor"
import { ScopeToggle } from "~/features/meetings/components/scope-toggle"
import { useT, useLocale } from "~/shared/i18n/context"
import { toastError } from "~/shared/lib/toast"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  meeting: Meeting
}

export function MeetingEditDialog({ open, onOpenChange, meeting }: Props) {
  const t = useT()
  const locale = useLocale()
  const update = useUpdateMeeting()
  const changeEnd = useChangeSeriesEnd()
  const [type, setType] = useState(meeting.type)
  const [date, setDate] = useState(meeting.date)
  const [start, setStart] = useState(meeting.start)
  const [end, setEnd] = useState(meeting.end)
  const [desc, setDesc] = useState(meeting.desc)
  const [scope, setScope] = useState<MeetingMutationScope>("this")
  const [seriesUntil, setSeriesUntil] = useState(
    (meeting.recurrence_until ?? "").slice(0, 10)
  )

  const series = isSeriesMeeting(meeting)
  const lockDate = series && scope === "whole"

  useEffect(() => {
    if (open) {
      setType(meeting.type)
      setDate(meeting.date)
      setStart(meeting.start)
      setEnd(meeting.end)
      setDesc(meeting.desc)
      setScope("this")
      setSeriesUntil((meeting.recurrence_until ?? "").slice(0, 10))
    }
  }, [open, meeting])

  function onSave() {
    update.mutate(
      {
        id: meeting.id,
        scope,
        input: {
          type,
          start,
          end,
          desc,
          ...(lockDate ? {} : { date }),
        },
      },
      {
        onSuccess: () => {
          toast.success(t("meetings.edit.toastSuccess"))
          onOpenChange(false)
        },
        onError: (error) => toastError(error, t, "meetings.edit.toastError"),
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{t("meetings.edit.title")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          {series ? <ScopeToggle value={scope} onChange={setScope} /> : null}
          {series ? (
            <div className="flex flex-col gap-1.5">
              <Field label={t("meetings.edit.seriesEnds")}>
                <DatePicker
                  value={seriesUntil}
                  onChange={setSeriesUntil}
                  localeCode={locale}
                  placeholder={t("meetings.edit.seriesEnds")}
                />
              </Field>
              <Button
                type="button"
                variant="ghost"
                disabled={changeEnd.isPending || !seriesUntil}
                onClick={() =>
                  changeEnd.mutate(
                    { id: meeting.id, until: seriesUntil },
                    {
                      onSuccess: () =>
                        toast.success(t("meetings.edit.toastEndSuccess")),
                      onError: (error) =>
                        toastError(error, t, "meetings.edit.toastEndError"),
                    }
                  )
                }
              >
                {t("meetings.edit.updateEndDateBtn")}
              </Button>
            </div>
          ) : null}
          <Field label={t("meetings.edit.fieldTitle")}>
            <Input value={type} onChange={(e) => setType(e.target.value)} />
          </Field>
          <Field label={t("meetings.edit.fieldWhen")}>
            <MeetingWhenPicker
              value={{ date, start, end }}
              onChange={(next) => {
                setDate(next.date)
                setStart(next.start)
                setEnd(next.end)
              }}
              disabledDate={lockDate}
              localeCode={locale}
              labels={{
                date: lockDate
                  ? t("meetings.edit.fieldDateLocked")
                  : t("meetings.edit.fieldDate"),
                start: t("meetings.edit.fieldStart"),
                end: t("meetings.edit.fieldEnd"),
                hour: t("common.timeHour"),
                minute: t("common.timeMinute"),
              }}
            />
          </Field>
          <Field label={t("meetings.edit.fieldDescription")}>
            <Input value={desc} onChange={(e) => setDesc(e.target.value)} />
          </Field>
          <ParticipantsEditor
            meetingId={meeting.id}
            participants={meeting.participants}
            series={series}
          />
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={onSave} disabled={update.isPending}>
            {t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
    </div>
  )
}
