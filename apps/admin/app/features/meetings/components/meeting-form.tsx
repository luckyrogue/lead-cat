import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  CalendarPlus,
  Input,
  Label,
  Loader2,
  Pencil,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"
import { useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { MeetingRecurrence, MeetingScope } from "~/entities/meeting/types"
import { WEEKDAYS, toggleDay } from "~/features/meetings/lib/weekdays"

const RECURRENCES: MeetingRecurrence[] = [
  "once",
  "daily",
  "weekly",
  "monthly",
  "custom",
]

const RECURRENCE_LABELS: Record<MeetingRecurrence, string> = {
  once: "One-time",
  daily: "Daily",
  weekly: "Weekly",
  monthly: "Monthly",
  custom: "Custom days",
}

const schema = z
  .object({
    dept: z.string().min(1, "Department is required"),
    type: z.string().min(1, "Type is required"),
    host: z.string(),
    date: z.string().min(1, "Date is required"),
    start: z.string().min(1, "Start time is required"),
    end: z.string().min(1, "End time is required"),
    recurrence: z.enum(["once", "daily", "weekly", "monthly", "custom"]),
    recurrence_until: z.string(),
    recurrence_days: z.array(z.number().int().min(1).max(7)),
    participants: z.string(),
    desc: z.string(),
  })
  .refine((v) => v.end > v.start, {
    path: ["end"],
    message: "End must be after start",
  })
  .refine((v) => v.recurrence === "once" || v.recurrence_until.length > 0, {
    path: ["recurrence_until"],
    message: "Pick an end date for a repeating meeting",
  })
  .refine((v) => v.recurrence !== "custom" || v.recurrence_days.length > 0, {
    path: ["recurrence_days"],
    message: "Pick at least one weekday",
  })

export type MeetingFormValues = z.infer<typeof schema>

export type MeetingFormDefaults = Partial<MeetingFormValues>

type MeetingFormProps = {
  mode: "create" | "edit"
  pending: boolean
  series?: boolean
  defaults?: MeetingFormDefaults
  onSubmit: (values: MeetingFormValues, scope: MeetingScope) => void
}

const EMPTY: MeetingFormValues = {
  dept: "",
  type: "",
  host: "",
  date: "",
  start: "",
  end: "",
  recurrence: "once",
  recurrence_until: "",
  recurrence_days: [],
  participants: "",
  desc: "",
}

export function MeetingForm({
  mode,
  pending,
  series = false,
  defaults,
  onSubmit,
}: MeetingFormProps) {
  const {
    control,
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<MeetingFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { ...EMPTY, ...defaults },
  })

  const recurrence = watch("recurrence")
  const isEdit = mode === "edit"
  const showScope = isEdit && series
  const [scope, setScope] = useState<MeetingScope>("this")
  const lockDate = showScope && scope === "whole"

  return (
    <form
      onSubmit={handleSubmit((values) => onSubmit(values, scope))}
      className="flex flex-col gap-4"
      id="meeting-form"
    >
      {showScope ? (
        <Field label="Apply to">
          <Select
            value={scope}
            onValueChange={(value) => setScope(value as MeetingScope)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="this">This meeting only</SelectItem>
              <SelectItem value="whole">The whole series</SelectItem>
            </SelectContent>
          </Select>
        </Field>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Department" error={errors.dept?.message}>
          <Input placeholder="Engineering" {...register("dept")} />
        </Field>
        <Field label="Type" error={errors.type?.message}>
          <Input placeholder="Standup" {...register("type")} />
        </Field>
      </div>

      <Field label="Host" hint="Optional — defaults to you">
        <Input placeholder="Host name" {...register("host")} />
      </Field>

      <div className="grid gap-4 sm:grid-cols-3">
        <Field
          label="Date"
          hint={lockDate ? "Locked for series edits" : undefined}
          error={errors.date?.message}
        >
          <Input type="date" disabled={lockDate} {...register("date")} />
        </Field>
        <Field label="Start" error={errors.start?.message}>
          <Input type="time" {...register("start")} />
        </Field>
        <Field label="End" error={errors.end?.message}>
          <Input type="time" {...register("end")} />
        </Field>
      </div>

      {!isEdit ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Repeats">
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
                          {RECURRENCE_LABELS[r]}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
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
                              ? "rounded-[calc(var(--radius)*0.75)] border border-primary bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground"
                              : "rounded-[calc(var(--radius)*0.75)] border border-border bg-background px-3 py-1.5 text-sm text-foreground transition hover:bg-muted"
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
        </>
      ) : null}

      {!isEdit ? (
        <Field
          label="Participants"
          hint="Comma-separated emails"
          error={errors.participants?.message}
        >
          <Input
            placeholder="alice@company.com, bob@company.com"
            {...register("participants")}
          />
        </Field>
      ) : null}

      <Field label="Description" hint="Optional">
        <Input placeholder="Agenda or notes" {...register("desc")} />
      </Field>

      <div className="flex justify-end">
        <Button type="submit" disabled={pending}>
          {pending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : isEdit ? (
            <Pencil className="size-4" />
          ) : (
            <CalendarPlus className="size-4" />
          )}
          {isEdit ? "Save changes" : "Create meeting"}
        </Button>
      </div>
    </form>
  )
}

type FieldProps = {
  label: string
  hint?: string
  error?: string
  children: React.ReactNode
}

function Field({ label, hint, error, children }: FieldProps) {
  return (
    <div className="space-y-2">
      <Label>
        {label}
        {hint ? (
          <span className="ml-2 text-xs font-normal text-muted-foreground">
            {hint}
          </span>
        ) : null}
      </Label>
      {children}
      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}
