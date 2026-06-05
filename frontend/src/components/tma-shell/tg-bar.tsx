import { I18N } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon, tgIconBtn } from "@/shared/ui/cat/primitives"

export function TgBar({
  onLang,
  native = false,
}: {
  onLang: () => void
  /** Inside Telegram: no fake ⋮/✕ — native header already provides them. */
  native?: boolean
}) {
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
      {!native && (
        <>
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
        </>
      )}
    </div>
  )
}
