import { useCallback, useEffect, useMemo, useState } from "react"
import { Outlet, useRouterState } from "@tanstack/react-router"
import { useTheme } from "next-themes"
import { TmaAuthProvider, useTmaAuth } from "@/features/auth/auth-context"
import { isTelegramMiniApp } from "@/shared/tma/telegram-env"
import { useTelegramViewport } from "@/shared/tma/use-telegram-viewport"
import { TmaAppProvider } from "@/shared/tma/context"
import { DEFAULT_ACCENT, makePalette } from "@/shared/tma/palette"
import type { Lang } from "@/shared/tma/types"
import {
  LangDropdown,
  TabBar,
  TgBar,
  TmaFrame,
} from "@/components/tma-shell"

const OVERLAY_PREFIXES = [
  "/meetings/create",
  "/profile/colleague",
  "/profile/admin",
]

function isOverlayPath(pathname: string): boolean {
  return OVERLAY_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`)
  )
}

function TmaAuthGate() {
  const { status, errorCode, retry } = useTmaAuth()
  if (status === "authed") return <TmaAppShell />
  const botUsername = import.meta.env.VITE_BOT_USERNAME ?? ""
  const errorMessage =
    errorCode === "no_init_data"
      ? "Откройте приложение из Telegram."
      : errorCode === "invalid_init_data"
        ? "Сессия Telegram недействительна. Закройте и откройте Mini App заново."
        : "Не удалось войти. Проверьте, что backend запущен (make dev)."
  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 16,
        padding: 24,
        textAlign: "center",
      }}
    >
      {status === "loading" && <p>Загрузка…</p>}
      {status === "not_registered" && (
        <>
          <p>Сначала зарегистрируйтесь в боте командой /start.</p>
          {botUsername && (
            <a
              href={`https://t.me/${botUsername}?start`}
              target="_blank"
              rel="noreferrer"
            >
              Открыть бота
            </a>
          )}
        </>
      )}
      {status === "error" && (
        <>
          <p>{errorMessage}</p>
          <button type="button" onClick={retry}>
            Повторить
          </button>
        </>
      )}
    </div>
  )
}

function TmaAppShell() {
  const { resolvedTheme } = useTheme()
  const dark = resolvedTheme === "dark"
  const [accent] = useState(
    () => localStorage.getItem("lc-accent") || DEFAULT_ACCENT
  )
  const pal = useMemo(() => makePalette(dark, accent), [dark, accent])

  const [lang, setLangState] = useState<Lang>(
    () => (localStorage.getItem("lc-lang") as Lang) || "ru"
  )
  useEffect(() => {
    localStorage.setItem("lc-lang", lang)
  }, [lang])
  const setLang = useCallback((l: Lang) => setLangState(l), [])

  const [langOpen, setLangOpen] = useState(false)

  const ctxValue = useMemo(
    () => ({
      ...pal,
      dark,
      lang,
      setLang,
      openLangPicker: () => setLangOpen(true),
    }),
    [pal, dark, lang, setLang]
  )

  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const overlayOpen = isOverlayPath(pathname)
  const inTelegram = isTelegramMiniApp()

  return (
    <TmaAppProvider value={ctxValue}>
      <TmaFrame>
        <div className="tma-shell">
          {!overlayOpen && (
            <TgBar native={inTelegram} onLang={() => setLangOpen(true)} />
          )}
          <div className="tma-shell__main lc-scroll lc-screen">
            <Outlet />
          </div>
          {!overlayOpen && <TabBar />}
        </div>
        <LangDropdown open={langOpen} onClose={() => setLangOpen(false)} />
      </TmaFrame>
    </TmaAppProvider>
  )
}

export function TmaLayout() {
  useTelegramViewport()
  return (
    <TmaAuthProvider>
      <TmaAuthGate />
    </TmaAuthProvider>
  )
}
