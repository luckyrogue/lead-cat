import {
  DatePicker,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import type { MeetingListFilter } from "~/entities/meeting/types"
import { useT, useLocale } from "~/shared/i18n/context"

export type OrganizerOption = { id: string; label: string }

type MeetingsFilterBarProps = {
  filter: MeetingListFilter
  dept: string
  organizers: OrganizerOption[]
  onFilterChange: (patch: Partial<MeetingListFilter>) => void
  onDeptChange: (value: string) => void
}

export function MeetingsFilterBar({
  filter,
  dept,
  organizers,
  onFilterChange,
  onDeptChange,
}: MeetingsFilterBarProps) {
  const t = useT()
  const locale = useLocale()

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
      <Field label={t("meetings.filter.labelStatus")}>
        <Select
          value={filter.status ?? "all"}
          onValueChange={(value) =>
            onFilterChange({
              status:
                value === "all"
                  ? undefined
                  : (value as "scheduled" | "cancelled"),
            })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">
              {t("meetings.filter.statusAll")}
            </SelectItem>
            <SelectItem value="scheduled">
              {t("meetings.filter.statusScheduled")}
            </SelectItem>
            <SelectItem value="cancelled">
              {t("meetings.filter.statusCancelled")}
            </SelectItem>
          </SelectContent>
        </Select>
      </Field>

      <Field label={t("meetings.filter.labelOrganizer")}>
        <Select
          value={filter.organizer ?? "all"}
          onValueChange={(value) =>
            onFilterChange({ organizer: value === "all" ? undefined : value })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">
              {t("meetings.filter.organizerAnyone")}
            </SelectItem>
            {organizers.map((organizer) => (
              <SelectItem key={organizer.id} value={organizer.id}>
                {organizer.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field label={t("meetings.filter.labelFrom")}>
        <DatePicker
          value={filter.from ?? ""}
          onChange={(value) => onFilterChange({ from: value || undefined })}
          allowClear
          localeCode={locale}
          placeholder={t("meetings.filter.labelFrom")}
        />
      </Field>

      <Field label={t("meetings.filter.labelTo")}>
        <DatePicker
          value={filter.to ?? ""}
          onChange={(value) => onFilterChange({ to: value || undefined })}
          allowClear
          localeCode={locale}
          placeholder={t("meetings.filter.labelTo")}
        />
      </Field>

      <Field label={t("meetings.filter.labelDepartment")}>
        <Input
          placeholder={t("meetings.filter.deptPlaceholder")}
          value={dept}
          onChange={(event) => onDeptChange(event.target.value)}
        />
      </Field>
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
    <div className="min-w-0 space-y-2">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  )
}
