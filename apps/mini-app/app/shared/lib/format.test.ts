import { describe, expect, it } from "vitest"
import {
  formatDate,
  formatDateLong,
  formatTimeRange,
  addDaysIso,
} from "./format"

describe("formatDate", () => {
  it("formats a valid iso date in the given locale", () => {
    const out = formatDate("2026-06-20", "en")
    expect(out).toContain("Jun")
    expect(out).toContain("20")
  })
  it("returns the input unchanged for an invalid iso", () => {
    expect(formatDate("nope")).toBe("nope")
  })
})

describe("formatDateLong", () => {
  it("includes the year", () => {
    expect(formatDateLong("2026-06-20", "en")).toContain("2026")
  })
  it("returns the input unchanged for an invalid iso", () => {
    expect(formatDateLong("")).toBe("")
  })
})

describe("formatTimeRange", () => {
  it("joins start and end", () => {
    expect(formatTimeRange("09:00", "10:00")).toBe("09:00 – 10:00")
  })
  it("returns start alone when end is empty", () => {
    expect(formatTimeRange("09:00", "")).toBe("09:00")
  })
  it("returns empty string when start is empty", () => {
    expect(formatTimeRange("", "10:00")).toBe("")
  })
})

describe("addDaysIso", () => {
  it("adds days within a month", () => {
    expect(addDaysIso("2026-06-20", 5)).toBe("2026-06-25")
  })
  it("rolls over month boundaries", () => {
    expect(addDaysIso("2026-06-30", 1)).toBe("2026-07-01")
  })
  it("subtracts days", () => {
    expect(addDaysIso("2026-07-01", -1)).toBe("2026-06-30")
  })
})
