import {
  DateRangePicker,
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
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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

      <Field
        label={t("meetings.filter.labelDateRange")}
        className="sm:col-span-2"
      >
        <DateRangePicker
          value={{ from: filter.from ?? "", to: filter.to ?? "" }}
          onChange={(range) =>
            onFilterChange({
              from: range.from || undefined,
              to: range.to || undefined,
            })
          }
          allowClear
          localeCode={locale}
          placeholder={t("common.dateRangePlaceholder")}
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
  className,
}: {
  label: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={
        className ? `min-w-0 space-y-2 ${className}` : "min-w-0 space-y-2"
      }
    >
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  )
}
