import { describe, expect, it } from "vitest"
import { formatMeetingDate, formatDateTime, meetingTitle } from "./format"
import type { Meeting } from "~/entities/meeting/types"

const mk = (partial: Partial<Meeting>) => partial as unknown as Meeting

describe("formatMeetingDate", () => {
  it("formats a valid date with explicit locale + timeZone", () => {
    const out = formatMeetingDate("2026-06-20T09:00:00Z", {
      locale: "en-US",
      timeZone: "UTC",
    })
    expect(out).toContain("Jun")
    expect(out).toContain("20")
  })
  it("returns the em dash for an invalid date", () => {
    expect(formatMeetingDate("not-a-date")).toBe("—")
  })
})

describe("formatDateTime", () => {
  it("returns the em dash for an invalid date", () => {
    expect(formatDateTime("not-a-date")).toBe("—")
  })
})

describe("meetingTitle", () => {
  it("uses the trimmed type when present", () => {
    expect(meetingTitle(mk({ type: "  Standup  ", name: "X | Y" }), "fb")).toBe(
      "Standup",
    )
  })
  it("falls back to the second pipe segment of name", () => {
    expect(meetingTitle(mk({ type: "  ", name: "Team | Sync" }), "fb")).toBe("Sync")
  })
  it("falls back to the provided fallback when nothing usable", () => {
    expect(meetingTitle(mk({ type: "", name: "OnlyOne" }), "fb")).toBe("fb")
  })
})
