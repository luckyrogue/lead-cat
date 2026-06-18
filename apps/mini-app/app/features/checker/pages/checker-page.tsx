import { Button, DateRangePicker, Input, Label, toast } from "@leadcat/ui"
import { useState } from "react"

import { EmployeePicker } from "~/components/employee-picker"
import { PageHeader } from "~/components/page-header"
import { EmptyState, LoadingState } from "~/components/states"
import { FreeSlotList } from "~/features/checker/components/free-slot-list"
import { fetchFreeSlots } from "~/entities/meeting/api"
import type { Employee } from "~/entities/employee/types"
import type { FreeSlot } from "~/entities/meeting/types"
import { addDaysIso, todayIso } from "~/shared/lib/format"
import { useT, useLocale } from "~/shared/i18n/context"

export function CheckerPage() {
  const t = useT()
  const locale = useLocale()
  const [participants, setParticipants] = useState<Employee[]>([])
  const [from, setFrom] = useState(todayIso())
  const [to, setTo] = useState(addDaysIso(todayIso(), 6))
  const [duration, setDuration] = useState(30)
  const [slots, setSlots] = useState<FreeSlot[] | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  async function onCheck() {
    if (participants.length === 0) {
      toast.error(t("checker.toastNeedParticipants"))
      return
    }
    if (to < from || duration <= 0) {
      toast.error(t("checker.toastInvalidRange"))
      return
    }
    setLoading(true)
    setSlots(undefined)
    try {
      const result = await fetchFreeSlots({
        participants: participants.map((p) => p.email),
        from,
        to,
        duration_mins: duration,
      })
      setSlots(result)
    } catch {
      toast.error(t("checker.toastError"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t("checker.title")} subtitle={t("checker.subtitle")} />

      <div className="flex flex-col gap-1.5">
        <Label>{t("checker.participants")}</Label>
        <EmployeePicker selected={participants} onChange={setParticipants} />
      </div>

      <Field label={t("checker.fieldDateRange")}>
        <DateRangePicker
          value={{ from, to }}
          onChange={(range) => {
            setFrom(range.from)
            setTo(range.to)
          }}
          localeCode={locale}
          placeholder={t("common.dateRangePlaceholder")}
        />
      </Field>

      <Field label={t("checker.fieldDuration")}>
        <Input
          type="number"
          min={5}
          step={5}
          value={duration}
          onChange={(e) => setDuration(Number(e.target.value))}
        />
      </Field>

      <Button onClick={onCheck} disabled={loading}>
        {t("checker.findBtn")}
      </Button>

      {loading ? (
        <LoadingState label={t("checker.searching")} />
      ) : slots === undefined ? null : slots.length === 0 ? (
        <EmptyState
          title={t("checker.emptyTitle")}
          hint={t("checker.emptyHint")}
        />
      ) : (
        <FreeSlotList slots={slots} />
      )}
    </div>
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
