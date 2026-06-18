import * as React from "react"
import type { DateRange } from "react-day-picker"
import { CalendarDays, ChevronDown, X } from "lucide-react"

import {
  dayPickerLocaleFromCode,
  formatDateRangeLabel,
  formatIsoDate,
  type IsoDateRange,
  parseIsoDate,
} from "../../lib/date"
import { cn } from "../../lib/cn"
import { Calendar } from "./calendar"
import { pickerTriggerClassName } from "./date-picker"
import { Popover, PopoverContent, PopoverTrigger } from "./popover"

type DateRangePickerProps = {
  value?: IsoDateRange
  onChange?: (value: IsoDateRange) => void
  disabled?: boolean
  placeholder?: string
  className?: string
  id?: string
  localeCode?: string
  allowClear?: boolean
  fromDate?: Date
  toDate?: Date
}

function toSelectedRange(value: IsoDateRange | undefined): DateRange | undefined {
  const from = parseIsoDate(value?.from)
  const to = parseIsoDate(value?.to)
  if (!from && !to) {
    return undefined
  }
  return { from, to }
}

export function DateRangePicker({
  value = { from: "", to: "" },
  onChange,
  disabled = false,
  placeholder = "Pick a date range",
  className,
  id,
  localeCode,
  allowClear = false,
  fromDate,
  toDate,
}: DateRangePickerProps) {
  const [open, setOpen] = React.useState(false)
  const selected = toSelectedRange(value)
  const hasValue = Boolean(value.from || value.to)
  const label = hasValue
    ? formatDateRangeLabel(value, localeCode)
    : placeholder

  function handleSelect(range: DateRange | undefined) {
    if (!range) {
      return
    }
    const next: IsoDateRange = {
      from: range.from ? formatIsoDate(range.from) : "",
      to: range.to ? formatIsoDate(range.to) : "",
    }
    onChange?.(next)
    if (range.from && range.to) {
      setOpen(false)
    }
  }

  function handleClear(event: React.MouseEvent) {
    event.preventDefault()
    event.stopPropagation()
    onChange?.({ from: "", to: "" })
  }

  const calendarLocale = dayPickerLocaleFromCode(localeCode)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={disabled}>
        <button
          type="button"
          id={id}
          data-empty={!hasValue}
          className={cn(
            pickerTriggerClassName,
            "font-normal data-[empty=true]:text-muted-foreground",
            className
          )}
        >
          <span className="flex min-w-0 flex-1 items-center gap-2">
            <CalendarDays className="text-muted-foreground" />
            <span className="truncate text-left">{label}</span>
          </span>
          {allowClear && hasValue && !disabled ? (
            <span
              role="button"
              tabIndex={-1}
              aria-label="Clear date range"
              className="rounded-sm text-muted-foreground hover:text-foreground"
              onClick={handleClear}
            >
              <X className="size-4" />
            </span>
          ) : (
            <ChevronDown className="text-muted-foreground" />
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="range"
          locale={calendarLocale}
          selected={selected}
          defaultMonth={selected?.from ?? selected?.to}
          onSelect={handleSelect}
          disabled={[
            ...(fromDate ? [{ before: fromDate }] : []),
            ...(toDate ? [{ after: toDate }] : []),
          ]}
          numberOfMonths={1}
          autoFocus
        />
      </PopoverContent>
    </Popover>
  )
}
