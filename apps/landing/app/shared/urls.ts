function trimTrailingSlash(value: string): string {
  return value.trim().replace(/\/$/, "")
}

function botUsername(): string {
  const raw = import.meta.env.VITE_BOT_USERNAME
  if (typeof raw !== "string" || raw.trim() === "") {
    return ""
  }
  return raw.trim().replace(/^@/, "")
}

function adminBaseUrl(): string {
  const raw = import.meta.env.VITE_ADMIN_URL
  if (typeof raw === "string" && raw.trim() !== "") {
    return trimTrailingSlash(raw)
  }
  return "http://localhost:3001"
}

export function getAdminLoginUrl(): string {
  return `${adminBaseUrl()}/login`
}

/** Primary marketing CTA: open the Telegram bot, or admin login when bot username is unset. */
export function getStartedUrl(): string {
  const username = botUsername()
  if (username) {
    return `https://t.me/${username}`
  }
  return `${adminBaseUrl()}/login`
}

export function getStartedLinkProps(): {
  href: string
  target?: string
  rel?: string
} {
  const href = getStartedUrl()
  if (href.startsWith("https://t.me/")) {
    return { href, target: "_blank", rel: "noopener noreferrer" }
  }
  return { href }
}
