import {
  DEFAULT_ACCENT,
  hexToRgba,
  makePalette,
  type CatPalette,
} from "./palette"

export function catPaletteToCssVars(p: CatPalette): Record<string, string> {
  return {
    "--background": p.bg,
    "--foreground": p.text,
    "--card": p.card,
    "--card-foreground": p.text,
    "--popover": p.card,
    "--popover-foreground": p.text,
    "--primary": p.accent,
    "--primary-foreground": p.accentText,
    "--secondary": p.cardAlt,
    "--secondary-foreground": p.text,
    "--muted": p.cardAlt,
    "--muted-foreground": p.muted,
    "--accent": p.accentSoft,
    "--accent-foreground": p.text,
    "--destructive": p.danger,
    "--border": p.border,
    "--input": p.cardAlt,
    "--ring": p.accent,
    "--tab-bar-bg": p.tgBar,
    "--tma-bg": p.bg,
    "--tma-text": p.text,
    "--tma-hint": p.muted,
    "--tma-link": p.accent,
    "--tma-button": p.accent,
    "--tma-button-text": p.accentText,
    "--tma-secondary-bg": p.cardAlt,
    "--miniapp-pattern": p.pattern,
    "--miniapp-accent-glow": hexToRgba(p.accent, p.dark ? 0.35 : 0.45),
  }
}

export function applyCatPalette(frame: HTMLElement, palette: CatPalette): void {
  const vars = catPaletteToCssVars(palette)
  for (const [name, value] of Object.entries(vars)) {
    frame.style.setProperty(name, value)
  }
  frame.dataset.tmaTheme = palette.dark ? "dark" : "light"
  document.body.classList.add("tma-mode")
  if (palette.dark) {
    document.documentElement.classList.add("dark")
  } else {
    document.documentElement.classList.remove("dark")
  }
}

export function resolveTmaPalette(options?: {
  dark?: boolean
  accent?: string
}): CatPalette {
  const dark = options?.dark ?? false
  const accent = options?.accent ?? DEFAULT_ACCENT
  return makePalette(dark, accent)
}
