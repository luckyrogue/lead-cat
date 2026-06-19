import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Input,
  Label,
  Loader2,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { BookingEventType, EventTypeInput } from "~/entities/booking-event-type/types"
import { useT } from "~/shared/i18n/context"

const DURATION_OPTIONS = [15, 30, 45, 60] as const

const WEEKDAYS = [
  { value: 1, key: "1" },
  { value: 2, key: "2" },
  { value: 3, key: "3" },
  { value: 4, key: "4" },
  { value: 5, key: "5" },
  { value: 6, key: "6" },
  { value: 7, key: "7" },
] as const

const TIMEZONE_OPTIONS = [
  { value: "Asia/Almaty", label: "Almaty (UTC+5)" },
  { value: "Asia/Tashkent", label: "Tashkent (UTC+5)" },
  { value: "Asia/Bishkek", label: "Bishkek (UTC+6)" },
  { value: "Europe/Moscow", label: "Moscow (UTC+3)" },
  { value: "Europe/Kyiv", label: "Kyiv (UTC+2/3)" },
  { value: "Europe/London", label: "London (UTC+0/1)" },
  { value: "Asia/Dubai", label: "Dubai (UTC+4)" },
  { value: "Asia/Istanbul", label: "Istanbul (UTC+3)" },
  { value: "America/New_York", label: "New York (UTC-5/4)" },
  { value: "UTC", label: "UTC" },
]

function minutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
    .toString()
    .padStart(2, "0")
  const m = (minutes % 60).toString().padStart(2, "0")
  return `${h}:${m}`
}

function timeToMinutes(time: string): number {
  const [h, m] = time.split(":").map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return "UTC"
  }
}

const schema = z
  .object({
    title: z.string().min(1, "booking.errors.titleRequired"),
    description: z.string(),
    duration_mins: z.number().int().positive(),
    timezone: z.string().min(1, "booking.errors.timezoneRequired"),
    avail_weekdays: z
      .array(z.number().int().min(1).max(7))
      .min(1, "booking.errors.weekdayRequired"),
    avail_start_time: z.string().min(1, "booking.errors.startRequired"),
    avail_end_time: z.string().min(1, "booking.errors.endRequired"),
    active: z.boolean(),
  })
  .refine((v) => v.avail_end_time > v.avail_start_time, {
    path: ["avail_end_time"],
    message: "booking.errors.endAfterStart",
  })

type FormValues = z.infer<typeof schema>

function toFormValues(et: BookingEventType): FormValues {
  return {
    title: et.title,
    description: et.description,
    duration_mins: et.duration_mins,
    timezone: et.timezone,
    avail_weekdays: et.avail_weekdays,
    avail_start_time: minutesToTime(et.avail_start_minute),
    avail_end_time: minutesToTime(et.avail_end_minute),
    active: et.active,
  }
}

function toInput(values: FormValues): EventTypeInput {
  return {
    title: values.title,
    description: values.description,
    duration_mins: values.duration_mins,
    timezone: values.timezone,
    avail_weekdays: values.avail_weekdays,
    avail_start_minute: timeToMinutes(values.avail_start_time),
    avail_end_minute: timeToMinutes(values.avail_end_time),
    active: values.active,
  }
}

function toggleWeekday(days: number[], day: number): number[] {
  return days.includes(day) ? days.filter((d) => d !== day) : [...days, day]
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  pending: boolean
  editing: BookingEventType | null
  onSubmit: (input: EventTypeInput) => void
}

export function EventTypeDialog({
  open,
  onOpenChange,
  pending,
  editing,
  onSubmit,
}: Props) {
  const t = useT()

  const defaultValues: FormValues = editing
    ? toFormValues(editing)
    : {
        title: "",
        description: "",
        duration_mins: 30,
        timezone: browserTimezone(),
        avail_weekdays: [1, 2, 3, 4, 5],
        avail_start_time: "09:00",
        avail_end_time: "18:00",
        active: true,
      }

  const {
    register,
    control,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  function handleOpenChange(next: boolean) {
    if (!next) reset(defaultValues)
    onOpenChange(next)
  }

  function submit(values: FormValues) {
    onSubmit(toInput(values))
  }

  const te = (msg?: string) => (msg ? t(msg) : undefined)

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {editing ? t("booking.dialog.editTitle") : t("booking.dialog.createTitle")}
          </DialogTitle>
          <DialogDescription>
            {editing
              ? t("booking.dialog.editDescription")
              : t("booking.dialog.createDescription")}
          </DialogDescription>
        </DialogHeader>

        <form
          id="event-type-form"
          onSubmit={handleSubmit(submit)}
          className="flex flex-col gap-4"
        >
          <Field label={t("booking.fields.title")} error={te(errors.title?.message)}>
            <Input
              placeholder={t("booking.fields.titlePlaceholder")}
              {...register("title")}
            />
          </Field>

          <Field label={t("booking.fields.description")}>
            <textarea
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={t("booking.fields.descriptionPlaceholder")}
              {...register("description")}
            />
          </Field>

          <Field label={t("booking.fields.duration")} error={te(errors.duration_mins?.message)}>
            <Controller
              control={control}
              name="duration_mins"
              render={({ field }) => (
                <Select
                  value={String(field.value)}
                  onValueChange={(v) => field.onChange(Number(v))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DURATION_OPTIONS.map((d) => (
                      <SelectItem key={d} value={String(d)}>
                        {d} {t("booking.fields.minutes")}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </Field>

          <Field
            label={t("booking.fields.weekdays")}
            error={te(errors.avail_weekdays?.message)}
          >
            <Controller
              control={control}
              name="avail_weekdays"
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
                          field.onChange(toggleWeekday(field.value, day.value))
                        }
                        className={
                          active
                            ? "rounded-[calc(var(--radius)*0.75)] border border-primary bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground"
                            : "rounded-[calc(var(--radius)*0.75)] border border-border bg-background px-3 py-1.5 text-sm text-foreground transition hover:bg-muted"
                        }
                      >
                        {t(`meetings.form.weekdays.${day.key}`)}
                      </button>
                    )
                  })}
                </div>
              )}
            />
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field
              label={t("booking.fields.startTime")}
              error={te(errors.avail_start_time?.message)}
            >
              <Input type="time" {...register("avail_start_time")} />
            </Field>
            <Field
              label={t("booking.fields.endTime")}
              error={te(errors.avail_end_time?.message)}
            >
              <Input type="time" {...register("avail_end_time")} />
            </Field>
          </div>

          <Field label={t("booking.fields.timezone")} error={te(errors.timezone?.message)}>
            <Controller
              control={control}
              name="timezone"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TIMEZONE_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </Field>

          <Field label={t("booking.fields.active")}>
            <Controller
              control={control}
              name="active"
              render={({ field }) => (
                <button
                  type="button"
                  role="switch"
                  aria-checked={field.value}
                  onClick={() => field.onChange(!field.value)}
                  className={
                    field.value
                      ? "relative inline-flex h-6 w-11 items-center rounded-full border-2 border-primary bg-primary transition-colors"
                      : "relative inline-flex h-6 w-11 items-center rounded-full border-2 border-input bg-input transition-colors"
                  }
                >
                  <span
                    className={
                      field.value
                        ? "inline-block h-4 w-4 translate-x-5 rounded-full bg-primary-foreground transition-transform"
                        : "inline-block h-4 w-4 translate-x-0.5 rounded-full bg-background transition-transform"
                    }
                  />
                </button>
              )}
            />
          </Field>
        </form>

        <DialogFooter>
          <Button
            type="submit"
            form="event-type-form"
            disabled={pending}
          >
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {editing ? t("booking.dialog.save") : t("booking.dialog.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type FieldProps = {
  label: string
  error?: string
  children: React.ReactNode
}

function Field({ label, error, children }: FieldProps) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {children}
      {error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}
