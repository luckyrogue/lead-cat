import type { ReactNode } from "react"
import { I18N } from "@/shared/tma/i18n"
import type { Lang, TabKey } from "@/shared/tma/types"
import { hexToRgba } from "@/shared/tma/palette"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon, tgIconBtn } from "@/shared/ui/cat/primitives"
import { useMounted } from "./use-mounted"

export function TgBar({ onLang }: { onLang: () => void }) {
  const p = useTmaApp()
  return (
    <div
      style={{
        height: 52,
        flexShrink: 0,
        display: "flex",
        alignItems: "center",
        padding: "0 12px 0 16px",
        background: p.tgBar,
        position: "relative",
        zIndex: 6,
        borderBottom: `1px solid ${p.border}`,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          flex: 1,
          minWidth: 0,
        }}
      >
        <div
          style={{
            width: 34,
            height: 34,
            borderRadius: "50%",
            background: p.accent,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            flexShrink: 0,
            boxShadow: p.shadowSm,
          }}
        >
          <span style={{ fontSize: 19 }}>🐱</span>
        </div>
        <div style={{ minWidth: 0 }}>
          <div
            style={{
              fontFamily: "var(--font-display)",
              fontWeight: 800,
              fontSize: 16,
              color: p.tgBarText,
              lineHeight: 1.05,
            }}
          >
            Lead&nbsp;Cat
          </div>
          <div
            style={{
              fontSize: 11,
              color: p.muted,
              display: "flex",
              alignItems: "center",
              gap: 4,
            }}
          >
            <span
              style={{
                width: 6,
                height: 6,
                borderRadius: 3,
                background: p.ok,
                display: "inline-block",
              }}
            />
            mini app
          </div>
        </div>
      </div>
      <button
        type="button"
        onClick={onLang}
        style={tgIconBtn(p)}
        aria-label="language"
      >
        <span style={{ fontSize: 16 }}>{I18N[p.lang]._flag}</span>
        <CatIcon name="chevD" size={13} color={p.muted} sw={2.4} />
      </button>
      <button
        type="button"
        style={{ ...tgIconBtn(p), width: 34, gap: 0 }}
        aria-label="menu"
      >
        <span style={{ display: "flex", gap: 3 }}>
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              style={{
                width: 3.4,
                height: 3.4,
                borderRadius: 2,
                background: p.muted,
              }}
            />
          ))}
        </span>
      </button>
      <button
        type="button"
        style={{ ...tgIconBtn(p), width: 34 }}
        aria-label="close"
      >
        <CatIcon name="x" size={18} color={p.muted} sw={2.2} />
      </button>
    </div>
  )
}

export function LangDropdown({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(open, 220)
  if (!mounted) return null

  const langs: Lang[] = ["ru", "kk", "en"]
  return (
    <div
      style={{ position: "absolute", inset: 0, zIndex: 80 }}
      onClick={onClose}
    >
      <div
        style={{
          position: "absolute",
          inset: 0,
          background: shown ? "rgba(0,0,0,0.18)" : "rgba(0,0,0,0)",
          transition: "background .2s",
        }}
      />
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          position: "absolute",
          top: 100,
          right: 14,
          width: 220,
          background: p.card,
          borderRadius: 18,
          border: `1px solid ${p.border}`,
          boxShadow: p.shadow,
          overflow: "hidden",
          padding: 6,
          transformOrigin: "top right",
          transform: shown ? "scale(1)" : "scale(0.9)",
          opacity: shown ? 1 : 0,
          transition:
            "transform .2s cubic-bezier(.34,1.56,.64,1), opacity .18s",
        }}
      >
        <div
          style={{
            fontSize: 11,
            fontWeight: 700,
            color: p.faint,
            padding: "6px 10px 4px",
            letterSpacing: 0.5,
            textTransform: "uppercase",
          }}
        >
          {p.t("language")}
        </div>
        {langs.map((lng) => {
          const active = lng === p.lang
          return (
            <button
              key={lng}
              type="button"
              onClick={() => {
                p.setLang(lng)
                onClose()
              }}
              style={{
                width: "100%",
                display: "flex",
                alignItems: "center",
                gap: 11,
                padding: "11px 10px",
                borderRadius: 12,
                border: "none",
                cursor: "pointer",
                textAlign: "left",
                background: active ? p.accentSoft : "transparent",
                color: active ? p.accent : p.text,
                fontWeight: 700,
                fontSize: 15,
                fontFamily: "var(--font-display)",
              }}
            >
              <span style={{ fontSize: 20 }}>{I18N[lng]._flag}</span>
              <span style={{ flex: 1 }}>{I18N[lng]._label}</span>
              {active && (
                <CatIcon name="check" size={18} color={p.accent} sw={2.6} />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}

export function TabBar({
  tab,
  onTab,
  onCreate,
}: {
  tab: TabKey
  onTab: (t: TabKey) => void
  onCreate: () => void
}) {
  const p = useTmaApp()
  const items: (
    | {
        key: TabKey
        icon: "home" | "calendar" | "search" | "user"
        label: string
      }
    | { key: "_fab" }
  )[] = [
    { key: "home", icon: "home", label: p.t("nav_home") },
    { key: "meetings", icon: "calendar", label: p.t("nav_meetings") },
    { key: "_fab" },
    { key: "checker", icon: "search", label: p.t("nav_checker") },
    { key: "profile", icon: "user", label: p.t("nav_profile") },
  ]

  return (
    <div
      style={{
        flexShrink: 0,
        position: "relative",
        background: p.tgBar,
        borderTop: `1px solid ${p.border}`,
        paddingBottom: 22,
        paddingTop: 8,
        display: "flex",
        alignItems: "flex-start",
        zIndex: 6,
      }}
    >
      {items.map((it) => {
        if (it.key === "_fab") {
          return (
            <div
              key="_fab"
              style={{
                flex: 1,
                display: "flex",
                justifyContent: "center",
                position: "relative",
              }}
            >
              <button
                type="button"
                onClick={onCreate}
                style={{
                  position: "absolute",
                  top: -30,
                  width: 58,
                  height: 58,
                  borderRadius: 20,
                  background: p.accent,
                  border: `4px solid ${p.tgBar}`,
                  cursor: "pointer",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  boxShadow: `0 8px 20px ${hexToRgba(p.accent, 0.45)}`,
                  transition: "transform .15s",
                }}
                onPointerDown={(e) => {
                  e.currentTarget.style.transform = "scale(0.9) rotate(90deg)"
                }}
                onPointerUp={(e) => {
                  e.currentTarget.style.transform = "scale(1)"
                }}
                onPointerLeave={(e) => {
                  e.currentTarget.style.transform = "scale(1)"
                }}
              >
                <CatIcon name="plus" size={28} color={p.accentText} sw={2.6} />
              </button>
            </div>
          )
        }
        const active = it.key === tab
        return (
          <button
            key={it.key}
            type="button"
            onClick={() => onTab(it.key)}
            style={{
              flex: 1,
              background: "none",
              border: "none",
              cursor: "pointer",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              gap: 3,
              padding: 0,
              color: active ? p.accent : p.muted,
              transition: "color .18s",
            }}
          >
            <div
              style={{
                position: "relative",
                transition: "transform .2s cubic-bezier(.34,1.56,.64,1)",
                transform: active ? "translateY(-2px)" : "none",
              }}
            >
              <CatIcon
                name={it.icon}
                size={24}
                color={active ? p.accent : p.muted}
                sw={active ? 2.3 : 1.9}
              />
            </div>
            <span
              style={{
                fontSize: 10.5,
                fontWeight: active ? 800 : 600,
                fontFamily: "var(--font-display)",
              }}
            >
              {it.label}
            </span>
          </button>
        )
      })}
    </div>
  )
}

export function Sheet({
  open,
  onClose,
  children,
  maxH = "88%",
}: {
  open: boolean
  onClose: () => void
  children: ReactNode
  maxH?: string
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(open, 320)
  if (!mounted) return null

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 70,
        display: "flex",
        flexDirection: "column",
        justifyContent: "flex-end",
      }}
    >
      <div
        onClick={onClose}
        style={{
          position: "absolute",
          inset: 0,
          background: shown ? "rgba(20,12,4,0.42)" : "rgba(20,12,4,0)",
          transition: "background .3s",
          backdropFilter: shown ? "blur(2px)" : "none",
        }}
      />
      <div
        style={{
          position: "relative",
          background: p.bg,
          borderRadius: "26px 26px 0 0",
          maxHeight: maxH,
          display: "flex",
          flexDirection: "column",
          transform: shown ? "translateY(0)" : "translateY(102%)",
          transition: "transform .34s cubic-bezier(.32,.72,0,1)",
          boxShadow: "0 -10px 40px rgba(0,0,0,0.25)",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "center",
            paddingTop: 10,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 40,
              height: 5,
              borderRadius: 3,
              background: p.borderStrong,
            }}
          />
        </div>
        <div
          className="lc-scroll"
          style={{ overflow: "auto", padding: "8px 16px 26px" }}
        >
          {children}
        </div>
      </div>
    </div>
  )
}

export function Overlay({
  open,
  onClose,
  title,
  children,
  footer,
  onBack,
}: {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  footer?: ReactNode
  onBack?: () => void
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(open, 320)
  if (!mounted) return null

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        zIndex: 75,
        background: p.bg,
        display: "flex",
        flexDirection: "column",
        paddingTop: 54,
        transform: shown ? "translateX(0)" : "translateX(100%)",
        transition: "transform .32s cubic-bezier(.32,.72,0,1)",
        boxShadow: "-12px 0 40px rgba(0,0,0,0.12)",
      }}
    >
      <div
        style={{
          height: 52,
          flexShrink: 0,
          display: "flex",
          alignItems: "center",
          gap: 8,
          padding: "0 10px",
          borderBottom: `1px solid ${p.border}`,
          background: p.tgBar,
        }}
      >
        <button
          type="button"
          onClick={onBack ?? onClose}
          style={{ ...tgIconBtn(p), marginLeft: 0, width: 38 }}
        >
          <CatIcon
            name={onBack ? "chevL" : "x"}
            size={20}
            color={p.text}
            sw={2.2}
          />
        </button>
        <div
          style={{
            flex: 1,
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 17,
            color: p.text,
            textAlign: "center",
            marginRight: 38,
          }}
        >
          {title}
        </div>
      </div>
      <div className="lc-scroll" style={{ flex: 1, overflow: "auto" }}>
        {children}
      </div>
      {footer && (
        <div
          style={{
            flexShrink: 0,
            padding: "12px 16px 26px",
            borderTop: `1px solid ${p.border}`,
            background: p.tgBar,
          }}
        >
          {footer}
        </div>
      )}
    </div>
  )
}

export function TmaToast({
  data,
}: {
  data: { msg: string; emoji?: string } | null
}) {
  const p = useTmaApp()
  const { mounted, shown } = useMounted(!!data, 300)
  if (!mounted || !data) return null

  return (
    <div
      style={{
        position: "absolute",
        left: 0,
        right: 0,
        top: 60,
        display: "flex",
        justifyContent: "center",
        zIndex: 90,
        pointerEvents: "none",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 9,
          padding: "11px 16px",
          background: p.dark ? "#2A3A48" : "#2D2A26",
          color: "#fff",
          borderRadius: 14,
          fontWeight: 700,
          fontSize: 14,
          fontFamily: "var(--font-display)",
          boxShadow: "0 10px 30px rgba(0,0,0,0.3)",
          maxWidth: "84%",
          transform: shown
            ? "translateY(0) scale(1)"
            : "translateY(-16px) scale(0.92)",
          opacity: shown ? 1 : 0,
          transition: "all .3s cubic-bezier(.34,1.56,.64,1)",
        }}
      >
        <span style={{ fontSize: 17 }}>{data.emoji ?? "🐾"}</span>
        {data.msg}
      </div>
    </div>
  )
}

export function PawBurst({ show }: { show: boolean }) {
  if (!show) return null
  const bits = Array.from({ length: 14 })
  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        overflow: "hidden",
        zIndex: 95,
      }}
    >
      {bits.map((_, i) => {
        const ang = (i / bits.length) * Math.PI * 2
        const dist = 90 + (i % 4) * 36
        const hue = [25, 150, 300, 180, 95][i % 5]
        const style = {
          position: "absolute" as const,
          left: "50%",
          top: "42%",
          fontSize: 18 + (i % 3) * 6,
          color: `oklch(0.65 0.15 ${hue})`,
          animation: "lc-burst .9s cubic-bezier(.2,.7,.3,1) forwards",
          animationDelay: `${(i % 5) * 18}ms`,
          ["--tx" as string]: `${Math.cos(ang) * dist}px`,
          ["--ty" as string]: `${Math.sin(ang) * dist}px`,
          ["--rot" as string]: `${(i % 2 ? 1 : -1) * (120 + i * 12)}deg`,
        }
        return (
          <span key={i} style={style}>
            🐾
          </span>
        )
      })}
    </div>
  )
}

export function TmaFrame({ children }: { children: ReactNode }) {
  const p = useTmaApp()
  return (
    <div
      className="mx-auto flex min-h-[100dvh] w-full max-w-[480px] flex-col"
      style={{
        background: p.bg,
        backgroundImage: `radial-gradient(${p.pattern} 1.4px, transparent 1.4px)`,
        backgroundSize: "22px 22px",
        position: "relative",
      }}
    >
      {children}
    </div>
  )
}
