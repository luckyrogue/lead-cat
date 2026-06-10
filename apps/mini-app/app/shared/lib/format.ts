const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"]
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]

export function formatDate(iso: string): string {
  const d = parseDate(iso)
  if (!d) {
    return iso
  }
  return `${WEEKDAYS[d.getDay()]}, ${d.getDate()} ${MONTHS[d.getMonth()]}`
}

export function formatDateLong(iso: string): string {
  const d = parseDate(iso)
  if (!d) {
    return iso
  }
  return `${WEEKDAYS[d.getDay()]}, ${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}`
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
