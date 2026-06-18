/** Greeting name: Telegram first_name, else given name from bot FIO (Фамилия Имя …). */
export function getGreetingName(
  fullName: string | undefined,
  telegramFirstName?: string
): string {
  const fromTelegram = telegramFirstName?.trim()
  if (fromTelegram) {
    return fromTelegram
  }

  const parts = (fullName ?? "").trim().split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return parts[1]
  }
  if (parts.length === 1) {
    return parts[0]
  }
  return ""
}
