import type { CatPalette } from "./types"

export function hexLum(hex: string): number {
  const h = hex.replace("#", "")
  if (h.length < 6) return 0.5
  const r = parseInt(h.slice(0, 2), 16) / 255
  const g = parseInt(h.slice(2, 4), 16) / 255
  const b = parseInt(h.slice(4, 6), 16) / 255
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

export function onAccent(accent: string): string {
  return hexLum(accent) > 0.62 ? "#241a0e" : "#ffffff"
}

export function hexToRgba(hex: string, a: number): string {
  const h = hex.replace("#", "")
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  return `rgba(${r},${g},${b},${a})`
}

export function makePalette(dark: boolean, accent: string): CatPalette {
  const accentText = onAccent(accent)
  if (dark) {
    return {
      dark: true,
      accent,
      accentText,
      accentSoft: hexToRgba(accent, 0.18),
      accentLine: hexToRgba(accent, 0.4),
      bg: "#151E27",
      bg2: "#0F161D",
      card: "#1E2A35",
      cardAlt: "#243240",
      text: "#F4F6F8",
      muted: "rgba(244,246,248,0.56)",
      faint: "rgba(244,246,248,0.34)",
      border: "rgba(255,255,255,0.08)",
      borderStrong: "rgba(255,255,255,0.14)",
      tgBar: "#17212B",
      tgBarText: "#F4F6F8",
      shadow: "0 8px 30px rgba(0,0,0,0.45)",
      shadowSm: "0 2px 10px rgba(0,0,0,0.4)",
      pattern: "rgba(255,255,255,0.035)",
      danger: "#FF6B6B",
      dangerSoft: "rgba(255,107,107,0.15)",
      ok: "#4ADE80",
      okSoft: "rgba(74,222,128,0.16)",
    }
  }
  return {
    dark: false,
    accent,
    accentText,
    accentSoft: hexToRgba(accent, 0.12),
    accentLine: hexToRgba(accent, 0.32),
    bg: "#FFF8F0",
    bg2: "#FBF1E6",
    card: "#FFFFFF",
    cardAlt: "#FBF4EB",
    text: "#2D2A26",
    muted: "#8C8276",
    faint: "#BCB2A4",
    border: "#F0E2D4",
    borderStrong: "#E8D5C2",
    tgBar: "#FFFFFF",
    tgBarText: "#2D2A26",
    shadow: "0 12px 34px rgba(141,110,70,0.14)",
    shadowSm: "0 3px 12px rgba(141,110,70,0.10)",
    pattern: "rgba(232,123,53,0.07)",
    danger: "#E0533B",
    dangerSoft: "rgba(224,83,59,0.10)",
    ok: "#1F8A5B",
    okSoft: "rgba(31,138,91,0.12)",
  }
}

export const DEFAULT_ACCENT = "#E87B35"
