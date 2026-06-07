import type { CSSProperties } from "react"
import { hexToRgba } from "./palette"
import { paletteSurfaceVars } from "./surface-vars"
import type { CatPalette } from "./types"

export function paletteCssVars(p: CatPalette): CSSProperties {
  return {
    ...paletteSurfaceVars(p),
    "--tma-accent": p.accent,
    "--tma-accent-text": p.accentText,
    "--tma-accent-soft": p.accentSoft,
    "--tma-accent-line": p.accentLine,
    "--tma-bg": p.bg,
    "--tma-bg-2": p.bg2,
    "--tma-card": p.card,
    "--tma-card-alt": p.cardAlt,
    "--tma-text": p.text,
    "--tma-muted": p.muted,
    "--tma-faint": p.faint,
    "--tma-border": p.border,
    "--tma-border-strong": p.borderStrong,
    "--tma-tg-bar": p.tgBar,
    "--tma-tg-bar-text": p.tgBarText,
    "--tma-shadow": p.shadow,
    "--tma-shadow-sm": p.shadowSm,
    "--tma-pattern": p.pattern,
    "--tma-danger": p.danger,
    "--tma-danger-soft": p.dangerSoft,
    "--tma-ok": p.ok,
    "--tma-ok-soft": p.okSoft,
    "--tma-accent-glow": hexToRgba(p.accent, 0.45),
    "--tma-input-bg": p.dark ? p.cardAlt : "#ffffff",
    "--tma-segmented-track": p.dark
      ? "rgba(255,255,255,0.06)"
      : "#F2E6D8",
    "--tma-icon-btn-bg": p.dark ? "rgba(255,255,255,0.07)" : "#F2E9DE",
    "--tma-toggle-off": p.dark ? "rgba(255,255,255,0.16)" : "#E4D7C8",
    "--tma-toast-bg": p.dark ? "#2A3A48" : "#2D2A26",
  } as CSSProperties
}
