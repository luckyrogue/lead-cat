import { Button, Card, CardContent, CardHeader, CardTitle } from "@leadcat/ui"
import { useState } from "react"

interface Slot {
  start: string
  end: string
}

interface BookingResult {
  meet_link: string
  start: string
  end: string
}

interface Props {
  slug: string
  selectedSlot: Slot
  timezone: string
  onConflict: () => void
}

type FormState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "confirmed"; result: BookingResult }
  | { status: "conflict" }
  | { status: "badInput" }
  | { status: "error" }

function formatDateTime(iso: string, timezone: string): string {
  return new Intl.DateTimeFormat(undefined, {
    timeZone: timezone,
    weekday: "long",
    month: "long",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(iso))
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)
}

export function BookingForm({ slug, selectedSlot, timezone, onConflict }: Props) {
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [touched, setTouched] = useState({ name: false, email: false })
  const [formState, setFormState] = useState<FormState>({ status: "idle" })

  const nameError = touched.name && name.trim() === "" ? "Name is required." : null
  const emailError =
    touched.email && !isValidEmail(email) ? "A valid email is required." : null
  const canSubmit =
    formState.status !== "submitting" &&
    name.trim() !== "" &&
    isValidEmail(email)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setTouched({ name: true, email: true })
    if (!canSubmit) return

    setFormState({ status: "submitting" })
    try {
      const res = await fetch(`/api/book/${slug}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: name.trim(), email: email.trim(), start: selectedSlot.start }),
      })

      if (res.ok) {
        const result: BookingResult = await res.json()
        setFormState({ status: "confirmed", result })
        return
      }

      if (res.status === 409) {
        setFormState({ status: "conflict" })
        onConflict()
        return
      }

      if (res.status === 400) {
        setFormState({ status: "badInput" })
        return
      }

      setFormState({ status: "error" })
    } catch {
      setFormState({ status: "error" })
    }
  }

  if (formState.status === "confirmed") {
    const { result } = formState
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-xl">You&apos;re booked!</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {formatDateTime(result.start, timezone)}
          </p>
          <a
            href={result.meet_link}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
          >
            Join Google Meet
          </a>
          <p className="text-sm text-muted-foreground">
            Added to your calendar — check your email.
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <p className="text-sm font-medium">Your details</p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="booking-name" className="text-sm font-medium">
              Name
            </label>
            <input
              id="booking-name"
              type="text"
              autoComplete="name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={() => setTouched((t) => ({ ...t, name: true }))}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="Your name"
            />
            {nameError ? (
              <p className="text-xs text-destructive">{nameError}</p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <label htmlFor="booking-email" className="text-sm font-medium">
              Email
            </label>
            <input
              id="booking-email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onBlur={() => setTouched((t) => ({ ...t, email: true }))}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              placeholder="you@example.com"
            />
            {emailError ? (
              <p className="text-xs text-destructive">{emailError}</p>
            ) : null}
          </div>

          {formState.status === "badInput" ? (
            <p className="text-sm text-destructive">
              Please check your name and email.
            </p>
          ) : null}

          {formState.status === "error" ? (
            <p className="text-sm text-destructive">
              Something went wrong. Please try again.
            </p>
          ) : null}

          {formState.status === "conflict" ? (
            <p className="text-sm text-destructive">
              That time was just taken — please pick another.
            </p>
          ) : null}

          <Button
            type="submit"
            className="w-full"
            disabled={formState.status === "submitting"}
          >
            {formState.status === "submitting" ? "Booking..." : "Confirm booking"}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
