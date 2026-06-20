import { describe, expect, it } from "vitest"
import { translate } from "./translate"
import type { Dict } from "./types"

const active: Dict = { nav: { home: "Главная" }, hi: "Привет, {name}" }
const fallback: Dict = { nav: { home: "Home", about: "About" } }

describe("translate", () => {
  it("looks up a nested key in the active dict", () => {
    expect(translate(active, fallback, "nav.home")).toBe("Главная")
  })
  it("falls back when the key is missing in active", () => {
    expect(translate(active, fallback, "nav.about")).toBe("About")
  })
  it("returns the key itself when missing in both", () => {
    expect(translate(active, fallback, "nav.missing")).toBe("nav.missing")
  })
  it("interpolates params, leaving unknown placeholders intact", () => {
    expect(translate(active, fallback, "hi", { name: "Иван" })).toBe("Привет, Иван")
    expect(translate(active, fallback, "hi")).toBe("Привет, {name}")
  })
})
