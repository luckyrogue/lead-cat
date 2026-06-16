import { DEFAULT_LOCALE, type Locale } from "~/shared/i18n/types"

export function formatDate(
  iso: string,
  locale: Locale = DEFAULT_LOCALE
): string {
  const d = parseDate(iso)
  if (!d) {
    return iso
  }
  return new Intl.DateTimeFormat(locale, {
    weekday: "short",
    day: "numeric",
    month: "short",
  }).format(d)
}

export function formatDateLong(
  iso: string,
  locale: Locale = DEFAULT_LOCALE
): string {
  const d = parseDate(iso)
  if (!d) {
    return iso
  }
  return new Intl.DateTimeFormat(locale, {
    weekday: "short",
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(d)
}

export function formatTimeRange(start: string, end: string): string {
  if (!start) {
    return ""
  }
  return end ? `${start} – ${end}` : start
}

function parseDate(iso: string): Date | null {
  if (!iso) {
    return null
  }
  const [y, m, d] = iso.split("-").map(Number)
  if (!y || !m || !d) {
    return null
  }
  return new Date(y, m - 1, d)
}

export function todayIso(): string {
  const d = new Date()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${d.getFullYear()}-${m}-${day}`
}

export function addDaysIso(iso: string, days: number): string {
  const d = parseDate(iso) ?? new Date()
  d.setDate(d.getDate() + days)
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${d.getFullYear()}-${m}-${day}`
}

export function addMinutesToTime(time: string, minutes: number): string {
  const [h, m] = time.split(":").map(Number)
  if (Number.isNaN(h) || Number.isNaN(m)) {
    return time
  }
  const total = h * 60 + m + minutes
  const hh = Math.floor((total % (24 * 60)) / 60)
  const mm = total % 60
  return `${String(hh).padStart(2, "0")}:${String(mm).padStart(2, "0")}`
}
