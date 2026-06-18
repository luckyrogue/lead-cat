type TelegramWebAppUser = {
  id?: number
  first_name?: string
  last_name?: string
  username?: string
}

type TelegramWebApp = {
  initData?: string
  initDataUnsafe?: {
    user?: TelegramWebAppUser
  }
  ready?: () => void
  expand?: () => void
  colorScheme?: "light" | "dark"
  openLink?: (url: string) => void
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

export function getTelegramUser(): TelegramWebAppUser | undefined {
  return getWebApp()?.initDataUnsafe?.user
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

const INIT_DATA_POLL_MS = 50
const INIT_DATA_TIMEOUT_MS = 4_000

export function waitForInitData(
  timeoutMs = INIT_DATA_TIMEOUT_MS
): Promise<string> {
  if (typeof window === "undefined") {
    return Promise.resolve("")
  }

  return new Promise((resolve) => {
    const started = Date.now()

    const tick = () => {
      initTelegramViewport()
      const initData = getInitData()
      if (initData) {
        resolve(initData)
        return
      }
      if (Date.now() - started >= timeoutMs) {
        resolve("")
        return
      }
      window.setTimeout(tick, INIT_DATA_POLL_MS)
    }

    tick()
  })
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
