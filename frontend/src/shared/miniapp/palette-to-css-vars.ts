import type { CSSProperties } from "react"
import { hexToRgba } from "./palette"
import { paletteSurfaceVars } from "./surface-vars"
import type { CatPalette } from "./types"

export function paletteCssVars(p: CatPalette): CSSProperties {
  return {
    ...paletteSurfaceVars(p),
    "--miniapp-accent": p.accent,
    "--miniapp-accent-text": p.accentText,
    "--miniapp-accent-soft": p.accentSoft,
    "--miniapp-accent-line": p.accentLine,
    "--miniapp-bg": p.bg,
    "--miniapp-bg-2": p.bg2,
    "--miniapp-card": p.card,
    "--miniapp-card-alt": p.cardAlt,
    "--miniapp-text": p.text,
    "--miniapp-muted": p.muted,
    "--miniapp-faint": p.faint,
    "--miniapp-border": p.border,
    "--miniapp-border-strong": p.borderStrong,
    "--miniapp-tg-bar": p.tgBar,
    "--miniapp-tg-bar-text": p.tgBarText,
    "--miniapp-shadow": p.shadow,
    "--miniapp-shadow-sm": p.shadowSm,
    "--miniapp-pattern": p.pattern,
    "--miniapp-danger": p.danger,
    "--miniapp-danger-soft": p.dangerSoft,
    "--miniapp-ok": p.ok,
    "--miniapp-ok-soft": p.okSoft,
    "--miniapp-accent-glow": hexToRgba(p.accent, 0.45),
    "--miniapp-input-bg": p.dark ? p.cardAlt : "#ffffff",
    "--miniapp-segmented-track": p.dark ? "rgba(255,255,255,0.06)" : "#F2E6D8",
    "--miniapp-icon-btn-bg": p.dark ? "rgba(255,255,255,0.07)" : "#F2E9DE",
    "--miniapp-toggle-off": p.dark ? "rgba(255,255,255,0.16)" : "#E4D7C8",
    "--miniapp-toast-bg": p.dark ? "#2A3A48" : "#2D2A26",
  } as CSSProperties
}
