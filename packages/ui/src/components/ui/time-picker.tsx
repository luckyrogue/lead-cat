import * as React from "react"
import { Clock, ChevronDown } from "lucide-react"

import {
  formatTimeValue,
  parseTimeValue,
} from "../../lib/date"
import { cn } from "../../lib/cn"
import { pickerTriggerClassName } from "./date-picker"
import { Popover, PopoverContent, PopoverTrigger } from "./popover"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "./select"

type TimePickerProps = {
  value?: string
  onChange?: (value: string) => void
  disabled?: boolean
  placeholder?: string
  hourLabel?: string
  minuteLabel?: string
  className?: string
  id?: string
  minuteStep?: number
}

export function TimePicker({
  value = "",
  onChange,
  disabled = false,
  placeholder = "Pick a time",
  hourLabel = "Hour",
  minuteLabel = "Minute",
  className,
  id,
  minuteStep = 5,
}: TimePickerProps) {
  const [open, setOpen] = React.useState(false)

  function updateTime(nextHour: number, nextMinute: number) {
    onChange?.(formatTimeValue(nextHour, nextMinute))
  }

  const hours = React.useMemo(
    () => Array.from({ length: 24 }, (_, index) => String(index).padStart(2, "0")),
    []
  )
  const minutes = React.useMemo(() => {
    const items: string[] = []
    for (let minuteValue = 0; minuteValue < 60; minuteValue += minuteStep) {
      items.push(String(minuteValue).padStart(2, "0"))
    }
    return items
  }, [minuteStep])

  const current = parseTimeValue(value)
  const hourValue = String(current.hour).padStart(2, "0")
  const snappedMinute = Math.min(
    59,
    Math.round(current.minute / minuteStep) * minuteStep
  )
  const minuteValue = String(snappedMinute).padStart(2, "0")

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild disabled={disabled}>
        <button
          type="button"
          id={id}
          data-empty={!value}
          className={cn(
            pickerTriggerClassName,
            "font-normal data-[empty=true]:text-muted-foreground",
            className
          )}
        >
          <span className="flex min-w-0 flex-1 items-center gap-2">
            <Clock className="text-muted-foreground" />
            <span className="truncate text-left">
              {value ? value : placeholder}
            </span>
          </span>
          <ChevronDown className="text-muted-foreground" />
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-[min(100vw-2rem,16rem)] p-3" align="start">
        <div className="grid grid-cols-2 gap-2">
          <div className="space-y-1.5">
            <p className="text-xs font-medium text-muted-foreground">{hourLabel}</p>
            <Select
              value={hourValue}
              onValueChange={(nextHour) =>
                updateTime(Number(nextHour), Number(minuteValue))
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-56">
                {hours.map((hourOption) => (
                  <SelectItem key={hourOption} value={hourOption}>
                    {hourOption}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <p className="text-xs font-medium text-muted-foreground">{minuteLabel}</p>
            <Select
              value={minuteValue}
              onValueChange={(nextMinute) =>
                updateTime(Number(hourValue), Number(nextMinute))
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="max-h-56">
                {minutes.map((minuteOption) => (
                  <SelectItem key={minuteOption} value={minuteOption}>
                    {minuteOption}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
