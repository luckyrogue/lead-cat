import type { CSSProperties } from "react"
import { typeAccent } from "@/entities/meeting/constants"
import { hexToRgba } from "./palette"
import type { CatPalette } from "./types"

/** Per-hue icon/action tile (home quick actions, settings rows). */
export function hueSurfaceVars(hue: number, dark: boolean): CSSProperties {
  return {
    "--hue-deg": String(hue),
    "--hue-bg-l": String(dark ? 0.36 : 0.94),
    "--hue-bg-c": String(dark ? 0.08 : 0.06),
    "--hue-fg-l": String(dark ? 0.82 : 0.52),
    "--hue-fg-c": "0.15",
  } as CSSProperties
}

/** Per meeting-type accent (TypeTag, meeting card stripe). */
export function typeAccentVars(typeKey: string, dark: boolean): CSSProperties {
  const a = typeAccent(typeKey, dark)
  return {
    "--type-soft": a.soft,
    "--type-solid": a.solid,
    "--type-text": a.text,
  } as CSSProperties
}

/** Avatar size + generated hue. */
export function avatarVars(
  size: number,
  dark: boolean,
  hue: number
): CSSProperties {
  return {
    "--avatar-size": `${size}px`,
    "--avatar-fs": `${size * 0.38}px`,
    "--hue-deg": String(hue),
    "--hue-bg-l": String(dark ? 0.42 : 0.92),
    "--hue-bg-c": String(dark ? 0.09 : 0.07),
    "--hue-fg-l": String(dark ? 0.92 : 0.45),
    "--hue-fg-c": "0.13",
    "--avatar-ring": dark ? "#1E2A35" : "#fff",
  } as CSSProperties
}

/** Stack badge sizing for ParticipantStack "+N" pill. */
export function stackBadgeVars(size: number): CSSProperties {
  return {
    "--stack-size": `${size}px`,
    "--stack-fs": `${size * 0.34}px`,
  } as CSSProperties
}

/** Extra palette-derived surfaces set once on `.miniapp-frame`. */
export function paletteSurfaceVars(p: CatPalette): CSSProperties {
  return {
    "--miniapp-hero-gradient": `linear-gradient(135deg, ${hexToRgba(p.accent, p.dark ? 0.28 : 0.16)}, ${hexToRgba(p.accent, p.dark ? 0.1 : 0.04)})`,
    "--miniapp-hero-paw": hexToRgba(p.accent, p.dark ? 0.12 : 0.1),
    "--miniapp-auto-banner": `linear-gradient(135deg, oklch(${p.dark ? 0.34 : 0.95} 0.07 25), oklch(${p.dark ? 0.32 : 0.96} 0.05 45))`,
    "--miniapp-auto-paw": hexToRgba(p.accent, 0.12),
  } as CSSProperties
}
