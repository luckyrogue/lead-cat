import * as React from "react"
import type { Locale as DateFnsLocale } from "date-fns"
import { CalendarDays, ChevronDown, X } from "lucide-react"

import {
  dateFnsLocaleFromCode,
  dayPickerLocaleFromCode,
  formatDateLabel,
  formatIsoDate,
  parseIsoDate,
} from "../../lib/date"
import { cn } from "../../lib/cn"
import { Calendar } from "./calendar"
import { Popover, PopoverContent, PopoverTrigger } from "./popover"

export const pickerTriggerClassName =
  "flex h-10 min-w-0 w-full cursor-pointer items-center justify-between gap-2 rounded-[calc(var(--radius)*0.85)] border border-border/80 bg-input px-3.5 py-2 text-sm text-foreground shadow-sm transition-colors outline-none focus-visible:border-ring focus-visible:ring-4 focus-visible:ring-ring/25 disabled:cursor-not-allowed disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4"

type DatePickerProps = {
  value?: string
  onChange?: (value: string) => void
  disabled?: boolean
  placeholder?: string
  className?: string
  id?: string
  localeCode?: string
  allowClear?: boolean
  fromDate?: Date
  toDate?: Date
}

export function DatePicker({
  value = "",
  onChange,
  disabled = false,
  placeholder = "Pick a date",
  className,
  id,
  localeCode,
  allowClear = false,
  fromDate,
  toDate,
}: DatePickerProps) {
  const [open, setOpen] = React.useState(false)
  const locale = dateFnsLocaleFromCode(localeCode)
  const calendarLocale = dayPickerLocaleFromCode(localeCode)
  const selected = parseIsoDate(value)
  const hasValue = Boolean(value)

  function handleSelect(date: Date | undefined) {
    if (!date) {
      return
    }
    onChange?.(formatIsoDate(date))
    setOpen(false)
  }

  function handleClear(event: React.MouseEvent) {
    event.preventDefault()
    event.stopPropagation()
    onChange?.("")
  }

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
            <span className="truncate text-left">
              {hasValue ? formatDateLabel(value, locale) : placeholder}
            </span>
          </span>
          {allowClear && hasValue && !disabled ? (
            <span
              role="button"
              tabIndex={-1}
              aria-label="Clear date"
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
          mode="single"
          locale={calendarLocale}
          selected={selected}
          defaultMonth={selected}
          onSelect={handleSelect}
          disabled={[
            ...(fromDate ? [{ before: fromDate }] : []),
            ...(toDate ? [{ after: toDate }] : []),
          ]}
          initialFocus
        />
      </PopoverContent>
    </Popover>
  )
}
