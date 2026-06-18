import * as React from "react"

import {
  addMinutesToTime,
  DEFAULT_MEETING_DURATION_MIN,
  diffMinutes,
} from "../../lib/date"
import { cn } from "../../lib/cn"
import { DatePicker } from "./date-picker"
import { TimePicker } from "./time-picker"

export type MeetingWhenValue = {
  date: string
  start: string
  end: string
}

export type MeetingWhenLabels = {
  date?: string
  start?: string
  end?: string
}

type MeetingWhenPickerProps = {
  value: MeetingWhenValue
  onChange: (value: MeetingWhenValue) => void
  localeCode?: string
  disabledDate?: boolean
  disabledTime?: boolean
  labels?: MeetingWhenLabels
  className?: string
}

export function MeetingWhenPicker({
  value,
  onChange,
  localeCode,
  disabledDate = false,
  disabledTime = false,
  labels,
  className,
}: MeetingWhenPickerProps) {
  const prevStartRef = React.useRef(value.start)
  const endManuallySetRef = React.useRef(
    Boolean(value.end && value.start && value.end > value.start)
  )

  React.useEffect(() => {
    prevStartRef.current = value.start
  }, [value.start])

  function handleDateChange(date: string) {
    onChange({ ...value, date })
  }

  function handleStartChange(start: string) {
    const prevStart = prevStartRef.current
    let end = value.end

    if (!end || end <= start || !endManuallySetRef.current) {
      end = addMinutesToTime(start, DEFAULT_MEETING_DURATION_MIN)
      endManuallySetRef.current = false
    } else if (prevStart && prevStart !== start) {
      const duration = diffMinutes(prevStart, value.end)
      end = addMinutesToTime(start, duration)
    }

    prevStartRef.current = start
    onChange({ ...value, start, end })
  }

  function handleEndChange(end: string) {
    endManuallySetRef.current = true
    onChange({ ...value, end })
  }

  return (
    <div
      className={cn(
        "space-y-3 rounded-[calc(var(--radius)*1)] border border-border/80 bg-muted/20 p-3",
        className
      )}
    >
      <DatePicker
        value={value.date}
        onChange={handleDateChange}
        disabled={disabledDate}
        localeCode={localeCode}
        placeholder={labels?.date ?? "Date"}
      />
      <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-2">
        <TimePicker
          value={value.start}
          onChange={handleStartChange}
          disabled={disabledTime}
          placeholder={labels?.start ?? "Start"}
        />
        <span className="text-sm text-muted-foreground" aria-hidden="true">
          –
        </span>
        <TimePicker
          value={value.end}
          onChange={handleEndChange}
          disabled={disabledTime}
          placeholder={labels?.end ?? "End"}
        />
      </div>
    </div>
  )
}
