import { zodResolver } from "@hookform/resolvers/zod"
import { Button, ChevronLeft, Input, Label, toast } from "@leadcat/ui"
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
import { WEEKDAYS, toggleDay } from "~/features/meeting-create/lib/weekdays"
import { fetchConflicts } from "~/entities/meeting/api"
import { useCreateMeeting } from "~/entities/meeting/mutations"
import type { Employee } from "~/entities/employee/types"
import type { OccurrenceConflicts } from "~/entities/meeting/types"
import { addMinutesToTime, todayIso } from "~/shared/lib/format"

export function MeetingCreatePage() {
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
    control,
    formState: { errors },
  } = useForm<CreateMeetingForm>({
    resolver: zodResolver(createMeetingSchema),
    defaultValues: {
      type: "",
      dept: "",
      date: todayIso(),
      start: "10:00",
      end: addMinutesToTime("10:00", 30),
      recurrence: "once",
      recurrence_until: "",
      recurrence_days: [],
      desc: "",
    },
  })

  const recurrence = watch("recurrence")

  async function onCheckConflicts() {
    const v = getValues()
    if (participants.length === 0 || !v.date || !v.start || !v.end) {
      toast.error("Add participants and a time first")
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
      })
      setConflicts(result)
    } catch {
      toast.error("Couldn't check conflicts")
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
          toast.success("Meeting created")
          void navigate("/meetings")
        },
        onError: () => toast.error("Couldn't create meeting"),
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
        Back
      </button>
      <PageHeader title="New meeting" />

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <Field label="Title" error={errors.type?.message}>
          <Input placeholder="Weekly sync" {...register("type")} />
        </Field>
        <Field label="Department" error={errors.dept?.message}>
          <Input placeholder="Optional" {...register("dept")} />
        </Field>
        <Field label="Date" error={errors.date?.message}>
          <Input type="date" {...register("date")} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Start" error={errors.start?.message}>
            <Input type="time" {...register("start")} />
          </Field>
          <Field label="End" error={errors.end?.message}>
            <Input type="time" {...register("end")} />
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Repeats" error={errors.recurrence?.message}>
            <select
              className="h-10 rounded-[var(--radius)] border border-border bg-background px-3 text-sm"
              {...register("recurrence")}
            >
              <option value="once">One-time</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
              <option value="custom">Custom days</option>
            </select>
          </Field>
          {recurrence !== "once" ? (
            <Field
              label="Repeat until"
              error={errors.recurrence_until?.message}
            >
              <Input type="date" {...register("recurrence_until")} />
            </Field>
          ) : null}
        </div>

        {recurrence === "custom" ? (
          <Field label="On days" error={errors.recurrence_days?.message}>
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
                        {day.label}
                      </button>
                    )
                  })}
                </div>
              )}
            />
          </Field>
        ) : null}

        <div className="flex flex-col gap-1.5">
          <Label>Participants</Label>
          <EmployeePicker selected={participants} onChange={setParticipants} />
        </div>

        <Field label="Description" error={errors.desc?.message}>
          <Input placeholder="Optional" {...register("desc")} />
        </Field>

        <Button
          type="button"
          variant="outline"
          onClick={onCheckConflicts}
          disabled={checking}
        >
          Check conflicts
        </Button>
        <ConflictPreview loading={checking} occurrences={conflicts} />

        <Button type="submit" size="lg" disabled={create.isPending}>
          Create meeting
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
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  )
}
