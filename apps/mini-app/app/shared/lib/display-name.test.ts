import { describe, expect, it } from "vitest"
import { getGreetingName } from "./display-name"

describe("getGreetingName", () => {
  it("prefers a non-empty telegram first name", () => {
    expect(getGreetingName("Иванов Иван Иванович", "Vanya")).toBe("Vanya")
  })
  it("falls back to the given name (second FIO token)", () => {
    expect(getGreetingName("Иванов Иван Иванович")).toBe("Иван")
  })
  it("uses the single token when only one is present", () => {
    expect(getGreetingName("Иван")).toBe("Иван")
  })
  it("returns empty string for empty/undefined input", () => {
    expect(getGreetingName(undefined)).toBe("")
    expect(getGreetingName("   ", "  ")).toBe("")
  })
})
