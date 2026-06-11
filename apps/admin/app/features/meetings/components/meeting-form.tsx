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
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { MeetingRecurrence } from "~/entities/meeting/types"

const RECURRENCES: MeetingRecurrence[] = ["once", "daily", "weekly", "monthly"]

const schema = z
  .object({
    dept: z.string().min(1, "Department is required"),
    type: z.string().min(1, "Type is required"),
    host: z.string(),
    date: z.string().min(1, "Date is required"),
    start: z.string().min(1, "Start time is required"),
    end: z.string().min(1, "End time is required"),
    recurrence: z.enum(["once", "daily", "weekly", "monthly"]),
    recurrence_until: z.string(),
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

export type MeetingFormValues = z.infer<typeof schema>

export type MeetingFormDefaults = Partial<MeetingFormValues>

type MeetingFormProps = {
  mode: "create" | "edit"
  pending: boolean
  defaults?: MeetingFormDefaults
  onSubmit: (values: MeetingFormValues) => void
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
  participants: "",
  desc: "",
}

export function MeetingForm({
  mode,
  pending,
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

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col gap-4"
      id="meeting-form"
    >
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
        <Field label="Date" error={errors.date?.message}>
          <Input type="date" {...register("date")} />
        </Field>
        <Field label="Start" error={errors.start?.message}>
          <Input type="time" {...register("start")} />
        </Field>
        <Field label="End" error={errors.end?.message}>
          <Input type="time" {...register("end")} />
        </Field>
      </div>

      {!isEdit ? (
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
                        {r}
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
