import { describe, expect, it } from "vitest"

import { catPaletteToCssVars, resolveTmaPalette } from "./apply-cat-palette"
import { DEFAULT_ACCENT, makePalette } from "./palette"

describe("makePalette", () => {
  it("uses warm cream background in light mode", () => {
    const palette = makePalette(false, DEFAULT_ACCENT)
    expect(palette.bg).toBe("#FFF8F0")
    expect(palette.text).toBe("#2D2A26")
    expect(palette.accent).toBe("#E87B35")
  })

  it("uses readable light text in dark mode", () => {
    const palette = makePalette(true, DEFAULT_ACCENT)
    expect(palette.bg).toBe("#151E27")
    expect(palette.text).toBe("#F4F6F8")
    expect(palette.tgBar).toBe("#17212B")
  })
})

describe("catPaletteToCssVars", () => {
  it("maps palette to shadcn tokens on the frame", () => {
    const palette = makePalette(true, DEFAULT_ACCENT)
    const vars = catPaletteToCssVars(palette)
    expect(vars["--background"]).toBe("#151E27")
    expect(vars["--foreground"]).toBe("#F4F6F8")
    expect(vars["--primary"]).toBe("#E87B35")
    expect(vars["--tab-bar-bg"]).toBe("#17212B")
  })
})

describe("resolveTmaPalette", () => {
  it("defaults to light palette", () => {
    const palette = resolveTmaPalette()
    expect(palette.dark).toBe(false)
    expect(palette.bg).toBe("#FFF8F0")
  })
})
