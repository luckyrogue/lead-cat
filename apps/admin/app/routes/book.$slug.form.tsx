import { Button, Card, CardContent, CardHeader, CardTitle } from "@leadcat/ui"
import { useState } from "react"

import { useLocale, useT } from "~/shared/i18n/context"

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
  | { status: "conflict"; surveyToken?: string }
  | { status: "badInput"; surveyToken?: string }
  | { status: "error" }

function formatDateTime(iso: string, timezone: string, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
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

export function BookingForm({
  slug,
  selectedSlot,
  timezone,
  onConflict,
}: Props) {
  const t = useT()
  const locale = useLocale()
  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [touched, setTouched] = useState({ name: false, email: false })
  const [formState, setFormState] = useState<FormState>({ status: "idle" })

  const nameError =
    touched.name && name.trim() === ""
      ? t("publicBooking.errors.nameRequired")
      : null
  const emailError =
    touched.email && !isValidEmail(email)
      ? t("publicBooking.errors.emailInvalid")
      : null
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
        body: JSON.stringify({
          name: name.trim(),
          email: email.trim(),
          start: selectedSlot.start,
          language:
            typeof navigator !== "undefined"
              ? navigator.language.split("-")[0]
              : "",
        }),
      })

      if (res.ok) {
        const result: BookingResult = await res.json()
        setFormState({ status: "confirmed", result })
        return
      }

      if (res.status === 409) {
        const body = await res.json().catch(() => ({}))
        setFormState({ status: "conflict", surveyToken: body.survey_token })
        onConflict()
        return
      }

      if (res.status === 400) {
        const body = await res.json().catch(() => ({}))
        setFormState({ status: "badInput", surveyToken: body.survey_token })
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
          <CardTitle className="text-xl">
            {t("publicBooking.confirmed.title")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {formatDateTime(result.start, timezone, locale)}
          </p>
          <a
            href={result.meet_link}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
          >
            {t("publicBooking.confirmed.joinMeet")}
          </a>
          <p className="text-sm text-muted-foreground">
            {t("publicBooking.confirmed.calendarHint")}
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <p className="text-sm font-medium">{t("publicBooking.form.title")}</p>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="booking-name" className="text-sm font-medium">
              {t("publicBooking.form.nameLabel")}
            </label>
            <input
              id="booking-name"
              type="text"
              autoComplete="name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              onBlur={() => setTouched((prev) => ({ ...prev, name: true }))}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
              placeholder={t("publicBooking.form.namePlaceholder")}
            />
            {nameError ? (
              <p className="text-xs text-destructive">{nameError}</p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <label htmlFor="booking-email" className="text-sm font-medium">
              {t("publicBooking.form.emailLabel")}
            </label>
            <input
              id="booking-email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onBlur={() => setTouched((prev) => ({ ...prev, email: true }))}
              className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:ring-1 focus-visible:ring-ring focus-visible:outline-none"
              placeholder={t("publicBooking.form.emailPlaceholder")}
            />
            {emailError ? (
              <p className="text-xs text-destructive">{emailError}</p>
            ) : null}
          </div>

          {formState.status === "badInput" ? (
            <p className="text-sm text-destructive">
              {t("publicBooking.errors.badInput")}
            </p>
          ) : null}

          {formState.status === "error" ? (
            <p className="text-sm text-destructive">
              {t("publicBooking.error")}
            </p>
          ) : null}

          {formState.status === "conflict" ? (
            <p className="text-sm text-destructive">
              {t("publicBooking.errors.conflict")}
            </p>
          ) : null}

          {(formState.status === "conflict" ||
            formState.status === "badInput") &&
          formState.surveyToken ? (
            <a
              href={`/survey/${formState.surveyToken}`}
              className="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
            >
              {t("publicBooking.surveyCta")}
            </a>
          ) : null}

          <Button
            type="submit"
            className="w-full"
            disabled={formState.status === "submitting"}
          >
            {formState.status === "submitting"
              ? t("publicBooking.form.submitting")
              : t("publicBooking.form.submit")}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
