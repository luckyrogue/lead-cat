import { useEffect } from "react"

type SafeAreaInset = { top?: number; bottom?: number; left?: number; right?: number }

type TgWebApp = {
  ready: () => void
  expand: () => void
  viewportHeight?: number
  viewportStableHeight?: number
  safeAreaInset?: SafeAreaInset
  contentSafeAreaInset?: SafeAreaInset
  onEvent?: (event: string, cb: () => void) => void
  offEvent?: (event: string, cb: () => void) => void
}

function getWebApp(): TgWebApp | undefined {
  return (
    window as unknown as { Telegram?: { WebApp?: TgWebApp } }
  ).Telegram?.WebApp
}

function applyViewportHeight() {
  const tg = getWebApp()
  const h =
    tg?.viewportStableHeight ??
    tg?.viewportHeight ??
    window.visualViewport?.height ??
    window.innerHeight
  document.documentElement.style.setProperty("--tma-vh", `${h}px`)

  const safe = tg?.contentSafeAreaInset ?? tg?.safeAreaInset
  document.documentElement.style.setProperty(
    "--tma-safe-top",
    `${safe?.top ?? 0}px`
  )
  document.documentElement.style.setProperty(
    "--tma-safe-bottom",
    `${safe?.bottom ?? 0}px`
  )
}

/** Sync --tma-vh with Telegram WebApp viewport (stable height avoids keyboard jumps). */
export function useTelegramViewport() {
  useEffect(() => {
    document.body.classList.add("tma-mode")
    const tg = getWebApp()
    tg?.ready()
    tg?.expand()
    applyViewportHeight()

    const onResize = () => applyViewportHeight()
    window.addEventListener("resize", onResize)
    tg?.onEvent?.("viewportChanged", onResize)

    return () => {
      window.removeEventListener("resize", onResize)
      tg?.offEvent?.("viewportChanged", onResize)
      document.body.classList.remove("tma-mode")
      document.documentElement.style.removeProperty("--tma-vh")
    }
  }, [])
}
