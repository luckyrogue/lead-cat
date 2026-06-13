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
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Field label="Status">
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
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="scheduled">Scheduled</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
          </SelectContent>
        </Select>
      </Field>

      <Field label="Organizer">
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
            <SelectItem value="all">Anyone</SelectItem>
            {organizers.map((organizer) => (
              <SelectItem key={organizer.id} value={organizer.id}>
                {organizer.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field label="From">
        <Input
          type="date"
          value={filter.from ?? ""}
          onChange={(event) =>
            onFilterChange({ from: event.target.value || undefined })
          }
        />
      </Field>

      <Field label="To">
        <Input
          type="date"
          value={filter.to ?? ""}
          onChange={(event) =>
            onFilterChange({ to: event.target.value || undefined })
          }
        />
      </Field>

      <Field label="Department">
        <Input
          placeholder="Filter by department"
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
