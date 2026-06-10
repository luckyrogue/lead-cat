type TelegramWebApp = {
  initData?: string
  ready?: () => void
  expand?: () => void
  colorScheme?: "light" | "dark"
}

type TelegramGlobal = {
  WebApp?: TelegramWebApp
}

function getTelegram(): TelegramGlobal | undefined {
  if (typeof window === "undefined") {
    return undefined
  }
  return (window as unknown as { Telegram?: TelegramGlobal }).Telegram
}

export function getWebApp(): TelegramWebApp | undefined {
  return getTelegram()?.WebApp
}

export function getInitData(): string {
  const fromTelegram = getWebApp()?.initData
  if (fromTelegram && fromTelegram.length > 0) {
    return fromTelegram
  }
  const devTgId = import.meta.env.VITE_TMA_DEV_TG_ID
  if (typeof devTgId === "string" && devTgId.length > 0) {
    return devTgId
  }
  return ""
}

export function initTelegramViewport(): void {
  const webApp = getWebApp()
  if (!webApp) {
    return
  }
  try {
    webApp.ready?.()
    webApp.expand?.()
  } catch {
    return
  }
}

export function getBotStartUrl(): string | null {
  const username = import.meta.env.VITE_BOT_USERNAME
  if (typeof username !== "string" || username.length === 0) {
    return null
  }
  return `https://t.me/${username.replace(/^@/, "")}?start=register`
}
