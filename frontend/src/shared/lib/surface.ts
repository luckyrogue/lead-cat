export type Surface = "telegram" | "web"

// detectSurface inspects the Telegram WebApp object; "telegram" only when initData is non-empty.
export function detectSurface(webApp: { initData?: string } | undefined | null): Surface {
  return webApp && typeof webApp.initData === "string" && webApp.initData.length > 0 ? "telegram" : "web"
}
