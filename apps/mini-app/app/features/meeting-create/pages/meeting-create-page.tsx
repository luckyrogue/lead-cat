import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  ChevronLeft,
  DatePicker,
  Input,
  Label,
  MeetingWhenPicker,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@leadcat/ui"
import { useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { useNavigate } from "react-router"

import { EmployeePicker } from "~/components/employee-picker"
import { PageHeader } from "~/components/page-header"
import { ConflictPreview } from "~/features/meeting-create/components/conflict-preview"
import {
  createMeetingSchema,
  type CreateMeetingForm,
} from "~/features/meeting-create/lib/schema"
import { WEEKDAYS, toggleDay } from "@leadcat/types"
import { fetchConflicts } from "~/entities/meeting/api"
import { useCreateMeeting } from "~/entities/meeting/mutations"
import type { Employee } from "~/entities/employee/types"
import type { OccurrenceConflicts } from "~/entities/meeting/types"
import { useT, useLocale } from "~/shared/i18n/context"
import { todayIso, addMinutesToTime } from "~/shared/lib/format"
import { toastError } from "~/shared/lib/toast"

const RECURRENCES = ["once", "daily", "weekly", "monthly", "custom"] as const

export function MeetingCreatePage() {
  const t = useT()
  const locale = useLocale()
  const navigate = useNavigate()
  const create = useCreateMeeting()
  const [participants, setParticipants] = useState<Employee[]>([])
  const [conflicts, setConflicts] = useState<OccurrenceConflicts[] | undefined>(
    undefined
  )
  const [checking, setChecking] = useState(false)

  const {
    register,
    handleSubmit,
    getValues,
    watch,
    setValue,
    control,
    formState: { errors },
  } = useForm<CreateMeetingForm>({
    resolver: zodResolver(createMeetingSchema),
    defaultValues: {
      type: "",
      dept: "",
      date: todayIso(),
      start: "10:00",
      end: addMinutesToTime("10:00", 60),
      recurrence: "once",
      recurrence_until: "",
      recurrence_days: [],
      desc: "",
    },
  })

  const recurrence = watch("recurrence")
  const meetingDate = watch("date")
  const meetingStart = watch("start")
  const meetingEnd = watch("end")
  const te = (message?: string) => (message ? t(message) : undefined)

  async function onCheckConflicts() {
    const v = getValues()
    if (participants.length === 0 || !v.date || !v.start || !v.end) {
      toast.error(t("create.toastNeedParticipants"))
      return
    }
    setChecking(true)
    setConflicts(undefined)
    try {
      const result = await fetchConflicts({
        participants: participants.map((p) => p.email),
        date: v.date,
        start: v.start,
        end: v.end,
        recurrence: v.recurrence,
        recurrence_until:
          v.recurrence === "once" ? undefined : v.recurrence_until,
        recurrence_days:
          v.recurrence === "custom" ? v.recurrence_days : undefined,
      })
      setConflicts(result)
    } catch (error) {
      toastError(error, t, "create.toastConflictError")
    } finally {
      setChecking(false)
    }
  }

  function onSubmit(values: CreateMeetingForm) {
    create.mutate(
      {
        type: values.type,
        dept: values.dept,
        host: "",
        date: values.date,
        start: values.start,
        end: values.end,
        recurrence: values.recurrence,
        recurrence_until:
          values.recurrence === "once" ? undefined : values.recurrence_until,
        recurrence_days:
          values.recurrence === "custom" ? values.recurrence_days : undefined,
        desc: values.desc,
        participants: participants.map((p) => p.email),
      },
      {
        onSuccess: () => {
          toast.success(t("create.toastSuccess"))
          void navigate("/meetings")
        },
        onError: (error) => toastError(error, t, "create.toastError"),
      }
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <button
        type="button"
        onClick={() => navigate(-1)}
        className="-ml-1 flex w-fit items-center gap-1 text-sm font-medium text-muted-foreground"
      >
        <ChevronLeft className="size-4" />
        {t("common.back")}
      </button>
      <PageHeader title={t("create.pageTitle")} />

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <Field label={t("create.fieldTitle")} error={te(errors.type?.message)}>
          <Input
            placeholder={t("create.titlePlaceholder")}
            {...register("type")}
          />
        </Field>
        <Field label={t("create.fieldDept")} error={te(errors.dept?.message)}>
          <Input
            placeholder={t("create.deptPlaceholder")}
            {...register("dept")}
          />
        </Field>
        <Field
          label={t("create.fieldWhen")}
          error={
            te(errors.date?.message) ??
            te(errors.start?.message) ??
            te(errors.end?.message)
          }
        >
          <MeetingWhenPicker
            value={{
              date: meetingDate,
              start: meetingStart,
              end: meetingEnd,
            }}
            onChange={(next) => {
              setValue("date", next.date, { shouldValidate: true })
              setValue("start", next.start, { shouldValidate: true })
              setValue("end", next.end, { shouldValidate: true })
            }}
            localeCode={locale}
            labels={{
              date: t("create.fieldDate"),
              start: t("create.fieldStart"),
              end: t("create.fieldEnd"),
              hour: t("common.timeHour"),
              minute: t("common.timeMinute"),
            }}
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field
            label={t("create.fieldRepeats")}
            error={te(errors.recurrence?.message)}
          >
            <Controller
              control={control}
              name="recurrence"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {RECURRENCES.map((r) => (
                      <SelectItem key={r} value={r}>
                        {t(`create.recurrence.${r}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </Field>
          {recurrence !== "once" ? (
            <Field
              label={t("create.fieldRepeatUntil")}
              error={te(errors.recurrence_until?.message)}
            >
              <Controller
                control={control}
                name="recurrence_until"
                render={({ field }) => (
                  <DatePicker
                    value={field.value}
                    onChange={field.onChange}
                    localeCode={locale}
                    placeholder={t("create.fieldRepeatUntil")}
                  />
                )}
              />
            </Field>
          ) : null}
        </div>

        {recurrence === "custom" ? (
          <Field
            label={t("create.fieldOnDays")}
            error={te(errors.recurrence_days?.message)}
          >
            <Controller
              control={control}
              name="recurrence_days"
              render={({ field }) => (
                <div className="flex flex-wrap gap-2">
                  {WEEKDAYS.map((day) => {
                    const active = field.value.includes(day.value)
                    return (
                      <button
                        key={day.value}
                        type="button"
                        aria-pressed={active}
                        onClick={() =>
                          field.onChange(toggleDay(field.value, day.value))
                        }
                        className={
                          active
                            ? "rounded-[var(--radius)] border border-primary bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground"
                            : "rounded-[var(--radius)] border border-border bg-background px-3 py-1.5 text-sm text-foreground"
                        }
                      >
                        {t(`create.weekdays.${day.value}`)}
                      </button>
                    )
                  })}
                </div>
              )}
            />
          </Field>
        ) : null}

        <div className="flex flex-col gap-1.5">
          <Label>{t("create.fieldParticipants")}</Label>
          <EmployeePicker selected={participants} onChange={setParticipants} />
        </div>

        <Field
          label={t("create.fieldDescription")}
          error={errors.desc?.message}
        >
          <Input
            placeholder={t("create.descPlaceholder")}
            {...register("desc")}
          />
        </Field>

        <Button
          type="button"
          variant="outline"
          onClick={onCheckConflicts}
          disabled={checking}
        >
          {t("create.checkConflictsBtn")}
        </Button>
        <ConflictPreview loading={checking} occurrences={conflicts} />

        <Button type="submit" size="lg" disabled={create.isPending}>
          {t("create.createBtn")}
        </Button>
      </form>
    </div>
  )
}

function Field({
  label,
  error,
  children,
}: {
  label: string
  error?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      {children}
      {error ? (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}
