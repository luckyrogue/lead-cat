import { describe, expect, it } from "vitest"
import {
  parseIsoDate,
  formatIsoDate,
  parseTimeValue,
  formatTimeValue,
  addMinutesToTime,
  timeToMinutes,
  diffMinutes,
  buildTimeOptions,
  dateFnsLocaleFromCode,
  dayPickerLocaleFromCode,
  DEFAULT_MEETING_DURATION_MIN,
} from "./date"

describe("parse/format iso date", () => {
  it("round-trips a valid iso date", () => {
    expect(formatIsoDate(parseIsoDate("2026-06-20")!)).toBe("2026-06-20")
  })
  it("returns undefined for empty or garbage input", () => {
    expect(parseIsoDate(undefined)).toBeUndefined()
    expect(parseIsoDate("not-a-date")).toBeUndefined()
  })
})

describe("time value parsing", () => {
  it("parses HH:MM", () => {
    expect(parseTimeValue("09:30")).toEqual({ hour: 9, minute: 30 })
  })
  it("falls back to 0:0 for empty or garbage", () => {
    expect(parseTimeValue(undefined)).toEqual({ hour: 0, minute: 0 })
    expect(parseTimeValue("xx:yy")).toEqual({ hour: 0, minute: 0 })
  })
  it("formats with zero-padding", () => {
    expect(formatTimeValue(9, 5)).toBe("09:05")
  })
  it("timeToMinutes", () => {
    expect(timeToMinutes("01:30")).toBe(90)
  })
})

describe("addMinutesToTime", () => {
  it("adds within the day", () => {
    expect(addMinutesToTime("09:00", 90)).toBe("10:30")
  })
  it("wraps past midnight", () => {
    expect(addMinutesToTime("23:30", 60)).toBe("00:30")
  })
  it("wraps negative", () => {
    expect(addMinutesToTime("00:15", -30)).toBe("23:45")
  })
})

describe("diffMinutes", () => {
  it("returns positive delta", () => {
    expect(diffMinutes("09:00", "10:00")).toBe(60)
  })
  it("falls back to default for non-positive or missing", () => {
    expect(diffMinutes("10:00", "09:00")).toBe(DEFAULT_MEETING_DURATION_MIN)
    expect(diffMinutes("", "10:00")).toBe(DEFAULT_MEETING_DURATION_MIN)
  })
})

describe("buildTimeOptions", () => {
  it("produces a full grid for the default step", () => {
    expect(buildTimeOptions().length).toBe((24 * 60) / 5)
    expect(buildTimeOptions()[0]).toBe("00:00")
  })
  it("honors a custom step", () => {
    expect(buildTimeOptions(60).length).toBe(24)
  })
})

describe("locale resolution", () => {
  it("maps ru and kk to the ru locale", () => {
    expect(dateFnsLocaleFromCode("ru").code).toBe("ru")
    expect(dateFnsLocaleFromCode("kk").code).toBe("ru")
    expect(dayPickerLocaleFromCode("kk").code).toBe("ru")
  })
  it("defaults to en", () => {
    expect(dateFnsLocaleFromCode(undefined).code).toBe("en-US")
    expect(dateFnsLocaleFromCode("en").code).toBe("en-US")
  })
})
