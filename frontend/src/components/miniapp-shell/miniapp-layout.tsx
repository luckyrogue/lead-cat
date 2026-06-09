import { useCallback, useEffect, useMemo, useState } from "react"
import { Outlet, useRouterState } from "@tanstack/react-router"
import { useTheme } from "next-themes"
import { MiniAppAuthProvider, useMiniAppAuth } from "@/shared/auth/auth-context"
import { isTelegramMiniApp } from "@/shared/miniapp/telegram-env"
import { useTelegramViewport } from "@/shared/miniapp/use-telegram-viewport"
import { MiniAppProvider } from "@/shared/miniapp/context"
import { DEFAULT_ACCENT, makePalette } from "@/shared/miniapp/palette"
import { translate } from "@/shared/miniapp/i18n"
import {
  readStoredLang,
  writeStoredLang,
} from "@/shared/miniapp/stored-lang"
import type { Lang } from "@/shared/miniapp/types"
import { LangDropdown, TabBar, TgBar, MiniAppFrame } from "@/components/miniapp-shell"

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

function MiniAppAuthGate() {
  const { status, errorCode, retry } = useMiniAppAuth()
  if (status === "authed") return <TmaAppShell />
  const lang = readStoredLang()
  const t = (key: Parameters<typeof translate>[1]) => translate(lang, key)
  const botUsername = import.meta.env.VITE_BOT_USERNAME ?? ""
  const errorMessage =
    errorCode === "no_init_data"
      ? t("auth_no_init_data")
      : errorCode === "invalid_init_data"
        ? t("auth_invalid_init_data")
        : t("auth_login_failed")
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-6 text-center">
      {status === "loading" && <p>{t("auth_loading")}</p>}
      {status === "not_registered" && (
        <>
          <p>{t("auth_not_registered")}</p>
          {botUsername && (
            <a
              href={`https://t.me/${botUsername}?start`}
              target="_blank"
              rel="noreferrer"
              className="text-primary underline"
            >
              {t("auth_open_bot")}
            </a>
          )}
        </>
      )}
      {status === "error" && (
        <>
          <p>{errorMessage}</p>
          <button
            type="button"
            onClick={retry}
            className="bg-primary cursor-pointer rounded-xl border-none px-[18px] py-2.5 font-bold text-white"
          >
            {t("auth_retry")}
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

  const [lang, setLangState] = useState<Lang>(readStoredLang)
  useEffect(() => {
    writeStoredLang(lang)
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
    <MiniAppProvider value={ctxValue}>
      <MiniAppFrame>
        <div className="miniapp-shell">
          {!overlayOpen && (
            <TgBar native={inTelegram} onLang={() => setLangOpen(true)} />
          )}
          <div className="miniapp-shell__main lc-scroll animate-lc-screen-in">
            <Outlet />
          </div>
          {!overlayOpen && <TabBar />}
        </div>
        <LangDropdown open={langOpen} onClose={() => setLangOpen(false)} />
      </MiniAppFrame>
    </MiniAppProvider>
  )
}

export function MiniAppLayout() {
  useTelegramViewport()
  return (
    <MiniAppAuthProvider>
      <MiniAppAuthGate />
    </MiniAppAuthProvider>
  )
}
