import { I18N } from "@/shared/tma/i18n"
import type { Lang } from "@/shared/tma/types"
import { useTmaApp } from "@/shared/tma/context"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { useMounted } from "./use-mounted"

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
