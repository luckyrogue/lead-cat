import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useTheme } from "next-themes"
import { LinkTelegramBanner } from "@/features/auth-link-telegram/link-telegram-banner"
import { TmaAuthProvider, useTmaAuth } from "@/shared/tma/auth-context"
import { TmaAppProvider, useTmaApp } from "@/shared/tma/context"
import { DEFAULT_ACCENT, makePalette } from "@/shared/tma/palette"
import type {
  Employee,
  FreeSlot,
  Lang,
  Meeting,
  MeetingDraft,
  OverlayState,
  TabKey,
} from "@/shared/tma/types"
import { INITIAL_MEETINGS, INITIAL_SCENARIOS } from "@/shared/tma/mock-data"
import {
  buildTitle,
  detailToDraft,
  draftToMeeting,
  fmtDate,
} from "@/shared/tma/meeting-utils"
import { translate } from "@/shared/tma/i18n"
import { CatBtn } from "@/shared/ui/cat/primitives"
import { CatIcon } from "@/shared/ui/cat/icon"
import {
  LangDropdown,
  Overlay,
  PawBurst,
  Sheet,
  TabBar,
  TgBar,
  TmaFrame,
  TmaToast,
} from "@/widgets/tma-shell/tma-shell"
import { HomeScreen } from "./screens/home-screen"
import { MeetingsScreen, MeetingDetail } from "./screens/meetings-screen"
import { CheckerScreen } from "./screens/checker-screen"
import { AutoScreen } from "./screens/auto-screen"
import {
  ProfileScreen,
  ColleagueSchedule,
  AdminPanel,
} from "./screens/profile-screen"
import { CreateWizard } from "./screens/create-wizard"

function SuccessView({
  m,
  onDone,
  onView,
}: {
  m: Meeting
  onDone: () => void
  onView: () => void
}) {
  const p = useTmaApp()
  return (
    <div style={{ textAlign: "center", padding: "10px 6px 6px" }}>
      <div
        style={{
          fontSize: 60,
          marginBottom: 8,
          animation: "lc-pop .5s cubic-bezier(.34,1.56,.64,1)",
        }}
      >
        🐱
      </div>
      <h2
        style={{
          margin: "0 0 6px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 24,
          color: p.text,
        }}
      >
        {p.t("created")}
      </h2>
      <p style={{ margin: "0 0 18px", color: p.muted, fontSize: 14.5 }}>
        {p.t("createdSub")}
      </p>
      <div
        style={{
          background: p.cardAlt,
          borderRadius: 16,
          padding: "13px 15px",
          marginBottom: 18,
          textAlign: "left",
          border: `1px solid ${p.border}`,
        }}
      >
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 15.5,
            color: p.text,
            lineHeight: 1.3,
          }}
        >
          {buildTitle(m)}
        </div>
        <div
          style={{
            marginTop: 8,
            display: "flex",
            alignItems: "center",
            gap: 10,
            color: p.muted,
            fontSize: 13,
            fontWeight: 600,
          }}
        >
          <span
            style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
          >
            <CatIcon name="calendar" size={14} color={p.muted} sw={2} />
            {fmtDate(m.date, p.lang)}
          </span>
          <span
            style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
          >
            <CatIcon name="clock" size={14} color={p.muted} sw={2} />
            {m.start}–{m.end}
          </span>
        </div>
      </div>
      <div style={{ display: "flex", gap: 10 }}>
        <CatBtn variant="outline" full onClick={onView}>
          {p.t("preview")}
        </CatBtn>
        <CatBtn variant="primary" full onClick={onDone}>
          OK 🐾
        </CatBtn>
      </div>
    </div>
  )
}

function TmaContent() {
  const [tab, setTab] = useState<TabKey>("home")
  const [meetings, setMeetings] = useState<Meeting[]>(INITIAL_MEETINGS)
  const [scenarios, setScenarios] = useState(INITIAL_SCENARIOS)
  const [reminders, setReminders] = useState(["15m"])
  const [remOn, setRemOn] = useState(true)
  const p = useTmaApp()

  const [overlay, setOverlay] = useState<OverlayState>(null)
  const [detail, setDetail] = useState<Meeting | null>(null)
  const [langOpen, setLangOpen] = useState(false)
  const [success, setSuccess] = useState<Meeting | null>(null)
  const [burst, setBurst] = useState(false)

  const openCreate = (initial?: Partial<MeetingDraft>) =>
    setOverlay({ type: "create", initial })

  const completeCreate = (m: MeetingDraft & { end: string }) => {
    const nm = draftToMeeting(m, `new-${Date.now()}`)
    setMeetings((arr) => [...arr, nm])
    setOverlay(null)
    setTab("meetings")
    setBurst(true)
    setTimeout(() => setBurst(false), 1100)
    setSuccess(nm)
  }

  const fromSlot = (slot: FreeSlot, people: Employee[]) => {
    setTab("home")
    openCreate({
      date: slot.iso,
      start: slot.start,
      dur: slot.mins >= 120 ? 60 : slot.mins,
      participants: people,
    })
  }

  const deleteMeeting = (id: string) => {
    setMeetings((arr) => arr.filter((m) => m.id !== id))
    setDetail(null)
    p.showToast(
      p.lang === "en"
        ? "Meeting deleted"
        : p.lang === "kk"
          ? "Кездесу жойылды"
          : "Встреча удалена",
      "🗑️"
    )
  }

  const toggleScenario = (id: string) =>
    setScenarios((arr) =>
      arr.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s))
    )

  const screens: Record<TabKey, React.ReactNode> = {
    home: (
      <HomeScreen
        meetings={meetings}
        onCreate={() => openCreate()}
        onMeeting={setDetail}
        onColleague={() => setOverlay({ type: "colleague" })}
        onTab={setTab}
      />
    ),
    meetings: <MeetingsScreen meetings={meetings} onMeeting={setDetail} />,
    checker: <CheckerScreen onCreateFromSlot={fromSlot} />,
    auto: <AutoScreen scenarios={scenarios} onToggle={toggleScenario} />,
    profile: (
      <ProfileScreen
        onLang={() => setLangOpen(true)}
        onAdmin={() => setOverlay({ type: "admin" })}
        reminders={reminders}
        setReminders={setReminders}
        remOn={remOn}
        setRemOn={setRemOn}
      />
    ),
  }

  return (
    <TmaFrame>
      <div
        style={{
          position: "relative",
          display: "flex",
          flex: 1,
          flexDirection: "column",
          minHeight: "100dvh",
        }}
      >
        <TgBar onLang={() => setLangOpen(true)} />
        <div className="px-2 pt-1">
          <LinkTelegramBanner />
        </div>
        <div
          key={tab}
          className="lc-scroll lc-screen"
          style={{ flex: 1, overflow: "auto" }}
        >
          {screens[tab]}
        </div>
        <TabBar tab={tab} onTab={setTab} onCreate={() => openCreate()} />
      </div>

      <Overlay
        open={overlay?.type === "create"}
        onClose={() => setOverlay(null)}
        title={translate(p.lang, "create")}
      >
        {overlay?.type === "create" && (
          <CreateWizard
            initial={overlay.initial}
            meetings={meetings}
            onComplete={completeCreate}
          />
        )}
      </Overlay>

      <Overlay
        open={overlay?.type === "colleague"}
        onClose={() => setOverlay(null)}
        title={translate(p.lang, "colleagueSchedule")}
      >
        {overlay?.type === "colleague" && (
          <ColleagueSchedule meetings={meetings} />
        )}
      </Overlay>

      <Overlay
        open={overlay?.type === "admin"}
        onClose={() => setOverlay(null)}
        title={translate(p.lang, "admin")}
      >
        {overlay?.type === "admin" && <AdminPanel meetings={meetings} />}
      </Overlay>

      <Sheet open={!!detail} onClose={() => setDetail(null)}>
        {detail && (
          <MeetingDetail
            m={detail}
            onEdit={() => {
              setDetail(null)
              openCreate(detailToDraft(detail))
            }}
            onDelete={() => deleteMeeting(detail.id)}
          />
        )}
      </Sheet>

      <Sheet open={!!success} onClose={() => setSuccess(null)} maxH="70%">
        {success && (
          <SuccessView
            m={success}
            onDone={() => setSuccess(null)}
            onView={() => {
              setSuccess(null)
              setDetail(success)
            }}
          />
        )}
      </Sheet>

      <LangDropdown open={langOpen} onClose={() => setLangOpen(false)} />
      <PawBurst show={burst} />
    </TmaFrame>
  )
}

function TmaAppInner() {
  const { resolvedTheme } = useTheme()
  const dark = resolvedTheme === "dark"

  useEffect(() => {
    document.body.classList.add("tma-mode")
    return () => document.body.classList.remove("tma-mode")
  }, [])
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

  const toastRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const [toast, setToast] = useState<{ msg: string; emoji?: string } | null>(
    null
  )

  const showToast = useCallback((msg: string, emoji?: string) => {
    clearTimeout(toastRef.current)
    setToast({ msg, emoji })
    toastRef.current = setTimeout(() => setToast(null), 2200)
  }, [])

  const ctxValue = useMemo(
    () => ({ ...pal, dark, lang, setLang, showToast }),
    [pal, dark, lang, setLang, showToast]
  )

  return (
    <TmaAppProvider value={ctxValue}>
      <TmaContent />
      <TmaToast data={toast} />
    </TmaAppProvider>
  )
}

export function TmaApp() {
  return (
    <TmaAuthProvider>
      <TmaAuthGate />
    </TmaAuthProvider>
  )
}

function TmaAuthGate() {
  const { status, retry } = useTmaAuth()
  if (status === "authed") return <TmaAppInner />
  const botUsername = import.meta.env.VITE_BOT_USERNAME ?? ""
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
          <p>Не удалось войти. Откройте приложение из Telegram.</p>
          <button onClick={retry}>Повторить</button>
        </>
      )}
    </div>
  )
}
