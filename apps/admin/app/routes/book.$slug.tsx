import { Card, CardContent, CardHeader, CardTitle, Loader2 } from "@leadcat/ui"
import { useEffect, useState } from "react"
import { useParams } from "react-router"
import { BookingForm } from "./book.$slug.form"

interface Slot {
  start: string
  end: string
}

interface BookingEvent {
  title: string
  description: string
  duration_mins: number
  org_name: string
  timezone: string
}

interface BookingData {
  event: BookingEvent
  slots: Slot[]
}

type PageState =
  | { status: "loading" }
  | { status: "notFound" }
  | { status: "error" }
  | { status: "loaded"; data: BookingData }

function formatDayLabel(isoStart: string, timezone: string): string {
  return new Intl.DateTimeFormat(undefined, {
    timeZone: timezone,
    weekday: "short",
    month: "short",
    day: "numeric",
  }).format(new Date(isoStart))
}

function formatTimeLabel(isoStart: string, timezone: string): string {
  return new Intl.DateTimeFormat(undefined, {
    timeZone: timezone,
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(isoStart))
}

function getTzLabel(timezone: string): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      timeZone: timezone,
      timeZoneName: "short",
    })
      .formatToParts(new Date())
      .find((p) => p.type === "timeZoneName")?.value ?? timezone
  } catch {
    return timezone
  }
}

function groupSlotsByDay(slots: Slot[], timezone: string): Map<string, Slot[]> {
  const map = new Map<string, Slot[]>()
  for (const slot of slots) {
    const day = formatDayLabel(slot.start, timezone)
    const existing = map.get(day)
    if (existing) {
      existing.push(slot)
    } else {
      map.set(day, [slot])
    }
  }
  return map
}

export default function BookSlugPage() {
  const { slug } = useParams<{ slug: string }>()
  const [state, setState] = useState<PageState>({ status: "loading" })
  const [selectedSlot, setSelectedSlot] = useState<Slot | null>(null)
  const [selectedDay, setSelectedDay] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [fetchKey, setFetchKey] = useState(0)

  useEffect(() => {
    if (!slug) return
    setState({ status: "loading" })
    setSelectedSlot(null)
    setSelectedDay(null)
    setShowForm(false)

    fetch(`/api/book/${slug}`)
      .then((res) => {
        if (res.status === 404) {
          setState({ status: "notFound" })
          return null
        }
        if (!res.ok) {
          setState({ status: "error" })
          return null
        }
        return res.json()
      })
      .then((json: BookingData | null) => {
        if (!json) return
        setState({ status: "loaded", data: json })
        const grouped = groupSlotsByDay(json.slots, json.event.timezone)
        const firstDay = grouped.keys().next().value ?? null
        setSelectedDay(firstDay)
      })
      .catch(() => setState({ status: "error" }))
  }, [slug, fetchKey])

  if (state.status === "loading") {
    return (
      <div className="flex min-h-svh items-center justify-center">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (state.status === "notFound") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="text-muted-foreground">This booking link isn't available.</p>
      </div>
    )
  }

  if (state.status === "error") {
    return (
      <div className="flex min-h-svh items-center justify-center p-6">
        <p className="text-destructive">Something went wrong. Please try again.</p>
      </div>
    )
  }

  const { event, slots } = state.data
  const grouped = groupSlotsByDay(slots, event.timezone)
  const days = Array.from(grouped.keys())
  const tzLabel = getTzLabel(event.timezone)
  const daySlots = selectedDay ? (grouped.get(selectedDay) ?? []) : []

  return (
    <div className="flex min-h-svh items-start justify-center px-4 pt-16 pb-10 sm:items-center sm:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,oklch(0.9_0.07_55/0.45),transparent_28%),radial-gradient(circle_at_bottom_right,oklch(0.93_0.08_92/0.6),transparent_36%)]" />
      <div className="relative w-full max-w-lg space-y-6">
        <Card>
          <CardHeader>
            <CardTitle className="text-xl">{event.title}</CardTitle>
            <div className="flex flex-wrap gap-3 text-sm text-muted-foreground">
              <span>{event.duration_mins} min</span>
              <span>&middot;</span>
              <span>{event.org_name}</span>
            </div>
            {event.description ? (
              <p className="mt-1 text-sm text-muted-foreground">{event.description}</p>
            ) : null}
          </CardHeader>
        </Card>

        {slots.length === 0 ? (
          <Card>
            <CardContent className="py-6">
              <p className="text-center text-sm text-muted-foreground">
                No available times in the next two weeks.
              </p>
            </CardContent>
          </Card>
        ) : (
          <>
            <Card>
              <CardHeader>
                <p className="text-sm font-medium">Select a day</p>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {days.map((day) => (
                    <button
                      key={day}
                      type="button"
                      onClick={() => {
                        setSelectedDay(day)
                        setSelectedSlot(null)
                      }}
                      className={[
                        "rounded-md border px-3 py-1.5 text-sm transition-colors",
                        day === selectedDay
                          ? "border-primary bg-primary text-primary-foreground"
                          : "border-border bg-background hover:bg-accent",
                      ].join(" ")}
                    >
                      {day}
                    </button>
                  ))}
                </div>
              </CardContent>
            </Card>

            {selectedDay ? (
              <Card>
                <CardHeader>
                  <p className="text-sm font-medium">
                    {selectedDay} &middot; <span className="text-muted-foreground">{tzLabel}</span>
                  </p>
                </CardHeader>
                <CardContent>
                  <div className="flex flex-wrap gap-2">
                    {daySlots.map((slot) => {
                      const timeLabel = formatTimeLabel(slot.start, event.timezone)
                      const isSelected = selectedSlot?.start === slot.start
                      return (
                        <button
                          key={slot.start}
                          type="button"
                          onClick={() => setSelectedSlot(isSelected ? null : slot)}
                          className={[
                            "rounded-md border px-3 py-1.5 text-sm transition-colors",
                            isSelected
                              ? "border-primary bg-primary text-primary-foreground"
                              : "border-border bg-background hover:bg-accent",
                          ].join(" ")}
                        >
                          {timeLabel}
                        </button>
                      )
                    })}
                  </div>
                </CardContent>
              </Card>
            ) : null}

            {selectedSlot && !showForm ? (
              <div className="space-y-2">
                <button
                  type="button"
                  onClick={() => setShowForm(true)}
                  className="flex h-9 w-full items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                >
                  Continue
                </button>
              </div>
            ) : null}

            {selectedSlot && showForm && slug ? (
              <BookingForm
                slug={slug}
                selectedSlot={selectedSlot}
                timezone={event.timezone}
                onConflict={() => setFetchKey((k) => k + 1)}
              />
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}
