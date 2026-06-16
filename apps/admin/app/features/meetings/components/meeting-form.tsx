import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  CalendarPlus,
  DatePicker,
  Input,
  Label,
  Loader2,
  Pencil,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  TimePicker,
} from "@leadcat/ui"
import { useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { MeetingRecurrence, MeetingScope } from "~/entities/meeting/types"
import { WEEKDAYS, toggleDay } from "@leadcat/types"
import { useT, useLocale } from "~/shared/i18n/context"

const RECURRENCES: MeetingRecurrence[] = [
  "once",
  "daily",
  "weekly",
  "monthly",
  "custom",
]

const schema = z
  .object({
    dept: z.string().min(1, "meetings.form.errors.deptRequired"),
    type: z.string().min(1, "meetings.form.errors.typeRequired"),
    host: z.string(),
    date: z.string().min(1, "meetings.form.errors.dateRequired"),
    start: z.string().min(1, "meetings.form.errors.startRequired"),
    end: z.string().min(1, "meetings.form.errors.endRequired"),
    recurrence: z.enum(["once", "daily", "weekly", "monthly", "custom"]),
    recurrence_until: z.string(),
    recurrence_days: z.array(z.number().int().min(1).max(7)),
    participants: z.string(),
    desc: z.string(),
  })
  .refine((v) => v.end > v.start, {
    path: ["end"],
    message: "meetings.form.errors.endAfterStart",
  })
  .refine((v) => v.recurrence === "once" || v.recurrence_until.length > 0, {
    path: ["recurrence_until"],
    message: "meetings.form.errors.repeatUntilRequired",
  })
  .refine((v) => v.recurrence !== "custom" || v.recurrence_days.length > 0, {
    path: ["recurrence_days"],
    message: "meetings.form.errors.weekdayRequired",
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
  const t = useT()
  const locale = useLocale()
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

  const te = (message?: string) => (message ? t(message) : undefined)
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
        <Field label={t("meetings.form.labelApplyTo")}>
          <Select
            value={scope}
            onValueChange={(value) => setScope(value as MeetingScope)}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="this">
                {t("meetings.form.scopeThis")}
              </SelectItem>
              <SelectItem value="whole">
                {t("meetings.form.scopeWhole")}
              </SelectItem>
            </SelectContent>
          </Select>
        </Field>
      ) : null}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field
          label={t("meetings.form.labelDepartment")}
          error={te(errors.dept?.message)}
        >
          <Input
            placeholder={t("meetings.form.placeholderDepartment")}
            {...register("dept")}
          />
        </Field>
        <Field
          label={t("meetings.form.labelType")}
          error={te(errors.type?.message)}
        >
          <Input
            placeholder={t("meetings.form.placeholderType")}
            {...register("type")}
          />
        </Field>
      </div>

      <Field
        label={t("meetings.form.labelHost")}
        hint={t("meetings.form.hintHost")}
      >
        <Input
          placeholder={t("meetings.form.placeholderHost")}
          {...register("host")}
        />
      </Field>

      <div className="grid gap-4 sm:grid-cols-3">
        <Field
          label={t("meetings.form.labelDate")}
          hint={lockDate ? t("meetings.form.hintDateLocked") : undefined}
          error={te(errors.date?.message)}
        >
          <Controller
            control={control}
            name="date"
            render={({ field }) => (
              <DatePicker
                value={field.value}
                onChange={field.onChange}
                disabled={lockDate}
                localeCode={locale}
                placeholder={t("meetings.form.labelDate")}
              />
            )}
          />
        </Field>
        <Field
          label={t("meetings.form.labelStart")}
          error={te(errors.start?.message)}
        >
          <Controller
            control={control}
            name="start"
            render={({ field }) => (
              <TimePicker
                value={field.value}
                onChange={field.onChange}
                placeholder={t("meetings.form.labelStart")}
              />
            )}
          />
        </Field>
        <Field
          label={t("meetings.form.labelEnd")}
          error={te(errors.end?.message)}
        >
          <Controller
            control={control}
            name="end"
            render={({ field }) => (
              <TimePicker
                value={field.value}
                onChange={field.onChange}
                placeholder={t("meetings.form.labelEnd")}
              />
            )}
          />
        </Field>
      </div>

      {!isEdit ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label={t("meetings.form.labelRepeats")}>
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
                          {t(`meetings.recurrence.${r}`)}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
            </Field>
            {recurrence !== "once" ? (
              <Field
                label={t("meetings.form.labelRepeatUntil")}
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
                      placeholder={t("meetings.form.labelRepeatUntil")}
                    />
                  )}
                />
              </Field>
            ) : null}
          </div>
          {recurrence === "custom" ? (
            <Field
              label={t("meetings.form.labelOnDays")}
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
                              ? "rounded-[calc(var(--radius)*0.75)] border border-primary bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground"
                              : "rounded-[calc(var(--radius)*0.75)] border border-border bg-background px-3 py-1.5 text-sm text-foreground transition hover:bg-muted"
                          }
                        >
                          {t(`meetings.form.weekdays.${day.value}`)}
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
          label={t("meetings.form.labelParticipants")}
          hint={t("meetings.form.hintParticipants")}
          error={errors.participants?.message}
        >
          <Input
            placeholder={t("meetings.form.placeholderParticipants")}
            {...register("participants")}
          />
        </Field>
      ) : null}

      <Field
        label={t("meetings.form.labelDescription")}
        hint={t("meetings.form.hintDescription")}
      >
        <Input
          placeholder={t("meetings.form.placeholderDescription")}
          {...register("desc")}
        />
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
          {isEdit
            ? t("meetings.form.submitSave")
            : t("meetings.form.submitCreate")}
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
