import { useEffect } from "react"

type SafeAreaInset = {
  top?: number
  bottom?: number
  left?: number
  right?: number
}

type TgWebApp = {
  ready: () => void
  expand: () => void
  isExpanded?: boolean
  disableVerticalSwipes?: () => void
  viewportHeight?: number
  viewportStableHeight?: number
  safeAreaInset?: SafeAreaInset
  contentSafeAreaInset?: SafeAreaInset
  onEvent?: (event: string, cb: () => void) => void
  offEvent?: (event: string, cb: () => void) => void
}

function getWebApp(): TgWebApp | undefined {
  return (window as unknown as { Telegram?: { WebApp?: TgWebApp } }).Telegram
    ?.WebApp
}

function applyViewportHeight() {
  const tg = getWebApp()
  const h =
    tg?.viewportStableHeight ??
    tg?.viewportHeight ??
    window.visualViewport?.height ??
    window.innerHeight
  document.documentElement.style.setProperty("--miniapp-vh", `${h}px`)

  const safe = tg?.contentSafeAreaInset ?? tg?.safeAreaInset
  document.documentElement.style.setProperty(
    "--miniapp-safe-top",
    `${safe?.top ?? 0}px`
  )
  document.documentElement.style.setProperty(
    "--miniapp-safe-bottom",
    `${safe?.bottom ?? 0}px`
  )
}

/** Sync --miniapp-vh with Telegram WebApp viewport (stable height avoids keyboard jumps). */
export function useTelegramViewport() {
  useEffect(() => {
    document.body.classList.add("miniapp-mode")
    const tg = getWebApp()
    tg?.ready()
    tg?.expand()
    // Stop the swipe-down-to-minimize gesture: it drags the whole webview and
    // pushes the top/bottom bars out of the visible area.
    tg?.disableVerticalSwipes?.()
    applyViewportHeight()

    const onViewport = () => {
      // Telegram collapses the app on swipe; re-expand so chrome stays in view.
      if (tg && tg.isExpanded === false) tg.expand()
      applyViewportHeight()
    }
    window.addEventListener("resize", onViewport)
    tg?.onEvent?.("viewportChanged", onViewport)
    tg?.onEvent?.("safeAreaChanged", applyViewportHeight)
    tg?.onEvent?.("contentSafeAreaChanged", applyViewportHeight)

    return () => {
      window.removeEventListener("resize", onViewport)
      tg?.offEvent?.("viewportChanged", onViewport)
      tg?.offEvent?.("safeAreaChanged", applyViewportHeight)
      tg?.offEvent?.("contentSafeAreaChanged", applyViewportHeight)
      document.body.classList.remove("miniapp-mode")
      document.documentElement.style.removeProperty("--miniapp-vh")
    }
  }, [])
}
