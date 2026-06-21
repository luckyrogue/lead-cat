import { describe, expect, it } from "vitest"

import { recurrenceLabel } from "./recurrence-label"

// translate() returns the key itself when no translation exists.
const t = (key: string) => (key === "create.recurrence.weekly" ? "Weekly" : key)

describe("recurrenceLabel", () => {
  it("returns an empty string for empty input or 'once'", () => {
    expect(recurrenceLabel(t, "")).toBe("")
    expect(recurrenceLabel(t, "once")).toBe("")
  })

  it("returns the translated label when the key exists", () => {
    expect(recurrenceLabel(t, "weekly")).toBe("Weekly")
  })

  it("falls back to the raw value when the key is untranslated", () => {
    expect(recurrenceLabel(t, "fortnightly")).toBe("fortnightly")
  })
})
