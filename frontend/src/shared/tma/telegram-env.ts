type TgWebApp = {
  initData?: string
  platform?: string
  close?: () => void
  BackButton?: {
    show: () => void
    hide: () => void
    onClick: (cb: () => void) => void
    offClick: (cb: () => void) => void
  }
}

export function getTelegramWebApp(): TgWebApp | undefined {
  return (
    window as unknown as { Telegram?: { WebApp?: TgWebApp } }
  ).Telegram?.WebApp
}

/** True when opened inside Telegram (native header is shown — skip mock chrome). */
export function isTelegramMiniApp(): boolean {
  const tg = getTelegramWebApp()
  return Boolean(tg?.initData || tg?.platform)
}
