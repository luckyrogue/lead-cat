import {
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import type { MeetingListFilter } from "~/entities/meeting/types"
import { useT } from "~/shared/i18n/context"

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

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
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
        <Input
          type="date"
          value={filter.from ?? ""}
          onChange={(event) =>
            onFilterChange({ from: event.target.value || undefined })
          }
        />
      </Field>

      <Field label={t("meetings.filter.labelTo")}>
        <Input
          type="date"
          value={filter.to ?? ""}
          onChange={(event) =>
            onFilterChange({ to: event.target.value || undefined })
          }
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
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  )
}
