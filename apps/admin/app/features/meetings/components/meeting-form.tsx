import { zodResolver } from "@hookform/resolvers/zod"
import {
  Button,
  CalendarPlus,
  DatePicker,
  Field,
  Input,
  Loader2,
  MeetingWhenPicker,
  Pencil,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  WeekdayPicker,
  addMinutesToTime,
  todayIso,
} from "@leadcat/ui"
import { useState } from "react"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import type { MeetingRecurrence, MeetingScope } from "~/entities/meeting/types"
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
    setValue,
    formState: { errors },
  } = useForm<MeetingFormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      ...EMPTY,
      ...(mode === "create"
        ? {
            date: todayIso(),
            start: "10:00",
            end: addMinutesToTime("10:00", 60),
          }
        : {}),
      ...defaults,
    },
  })

  const te = (message?: string) => (message ? t(message) : undefined)
  const recurrence = watch("recurrence")
  const meetingDate = watch("date")
  const meetingStart = watch("start")
  const meetingEnd = watch("end")
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

      <Field
        label={t("meetings.form.labelWhen")}
        hint={lockDate ? t("meetings.form.hintDateLocked") : undefined}
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
          disabledDate={lockDate}
          localeCode={locale}
          labels={{
            date: t("meetings.form.labelDate"),
            start: t("meetings.form.labelStart"),
            end: t("meetings.form.labelEnd"),
            hour: t("common.timeHour"),
            minute: t("common.timeMinute"),
          }}
        />
      </Field>

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
                  <WeekdayPicker
                    value={field.value}
                    onChange={field.onChange}
                    label={(day) => t(`meetings.form.weekdays.${day}`)}
                  />
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
