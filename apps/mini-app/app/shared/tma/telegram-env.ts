import { applyCatPalette, resolveTmaPalette } from "./apply-cat-palette"
import { DEFAULT_ACCENT } from "./palette"

type TelegramWebAppUser = {
  id?: number
  first_name?: string
  last_name?: string
  username?: string
}

type TelegramThemeParams = {
  bg_color?: string
  text_color?: string
  hint_color?: string
  link_color?: string
  button_color?: string
  button_text_color?: string
  secondary_bg_color?: string
}

type TelegramWebApp = {
  initData?: string
  initDataUnsafe?: {
    user?: TelegramWebAppUser
  }
  ready?: () => void
  expand?: () => void
  colorScheme?: "light" | "dark"
  themeParams?: TelegramThemeParams
  openLink?: (url: string) => void
  onEvent?: (event: string, cb: () => void) => void
  offEvent?: (event: string, cb: () => void) => void
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
  if (
    import.meta.env.DEV &&
    typeof devTgId === "string" &&
    devTgId.length > 0
  ) {
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

let themeListenerAttached = false

function resolveAccentFromTelegram(): string {
  return DEFAULT_ACCENT
}

export function syncTmaTheme(frame?: HTMLElement | null): void {
  if (typeof document === "undefined") {
    return
  }
  const target = frame ?? document.querySelector<HTMLElement>(".tma-frame")
  if (!target) {
    return
  }

  const webApp = getWebApp()
  const dark = webApp?.colorScheme === "dark"
  const palette = resolveTmaPalette({
    dark,
    accent: resolveAccentFromTelegram(),
  })
  applyCatPalette(target, palette)
}

function attachThemeListener(): void {
  if (themeListenerAttached) {
    return
  }
  const webApp = getWebApp()
  if (!webApp?.onEvent) {
    return
  }
  themeListenerAttached = true
  webApp.onEvent("themeChanged", () => {
    syncTmaTheme()
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
    syncTmaTheme()
    attachThemeListener()
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
