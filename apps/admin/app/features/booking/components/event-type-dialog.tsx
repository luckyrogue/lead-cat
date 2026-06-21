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
  Loader2,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"
import { Controller, useForm } from "react-hook-form"

import type {
  BookingEventType,
  EventTypeInput,
} from "~/entities/booking-event-type/types"
import { useSurveys } from "~/entities/survey/queries"
import { useT } from "~/shared/i18n/context"
import { getTimezoneOptions } from "~/shared/lib/timezone-options"

import { Field } from "./event-type-dialog-field"
import {
  DURATION_OPTIONS,
  WEEKDAYS,
  browserTimezone,
  schema,
  toFormValues,
  toInput,
  toggleWeekday,
  type FormValues,
} from "./event-type-dialog-helpers"

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  pending: boolean
  editing: BookingEventType | null
  onSubmit: (input: EventTypeInput) => void
  orgId: string | null
}

export function EventTypeDialog({
  open,
  onOpenChange,
  pending,
  editing,
  onSubmit,
  orgId,
}: Props) {
  const t = useT()
  const timezoneOptions = getTimezoneOptions(t)
  const { data: surveys } = useSurveys(orgId)

  const currentSurveyId = editing?.survey_id ?? null
  const surveyOptions = (surveys ?? []).filter(
    (s) => s.is_active || s.id === currentSurveyId
  )

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
        survey_id: null,
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
            {editing
              ? t("booking.dialog.editTitle")
              : t("booking.dialog.createTitle")}
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
          <Field
            label={t("booking.fields.title")}
            error={te(errors.title?.message)}
          >
            <Input
              placeholder={t("booking.fields.titlePlaceholder")}
              {...register("title")}
            />
          </Field>

          <Field label={t("booking.fields.description")}>
            <textarea
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={t("booking.fields.descriptionPlaceholder")}
              {...register("description")}
            />
          </Field>

          <Field
            label={t("booking.fields.duration")}
            error={te(errors.duration_mins?.message)}
          >
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

          <Field
            label={t("booking.fields.timezone")}
            error={te(errors.timezone?.message)}
          >
            <Controller
              control={control}
              name="timezone"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {timezoneOptions.map((opt) => (
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
                  aria-label={t("booking.fields.active")}
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

          <Field label={t("surveys.assignLabel")}>
            <Controller
              control={control}
              name="survey_id"
              render={({ field }) => (
                <Select
                  value={field.value ?? ""}
                  onValueChange={(v) => field.onChange(v === "" ? null : v)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t("surveys.assignNone")} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="">{t("surveys.assignNone")}</SelectItem>
                    {surveyOptions.map((s) => (
                      <SelectItem key={s.id} value={s.id}>
                        {s.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </Field>
        </form>

        <DialogFooter>
          <Button type="submit" form="event-type-form" disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {editing ? t("booking.dialog.save") : t("booking.dialog.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
