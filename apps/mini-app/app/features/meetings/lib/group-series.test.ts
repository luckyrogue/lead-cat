import { describe, expect, it } from "vitest"
import { groupBySeries } from "./group-series"
import type { Meeting } from "~/entities/meeting/types"

const mk = (id: string, series_id?: string) =>
  ({ id, series_id }) as unknown as Meeting

describe("groupBySeries", () => {
  it("returns [] for empty input", () => {
    expect(groupBySeries([])).toEqual([])
  })
  it("treats meetings without a series_id as singles", () => {
    const result = groupBySeries([mk("a"), mk("b")])
    expect(result).toHaveLength(2)
    expect(result.every((g) => g.kind === "single")).toBe(true)
  })
  it("groups by series in first-seen order, with singles appended after", () => {
    const result = groupBySeries([
      mk("s1a", "s1"),
      mk("solo"),
      mk("s2a", "s2"),
      mk("s1b", "s1"),
    ])
    expect(result.map((g) => g.kind)).toEqual(["series", "series", "single"])
    const first = result[0]
    if (first.kind !== "series") {
      throw new Error("expected first group to be a series")
    }
    expect(first.seriesId).toBe("s1")
    expect(first.meetings.map((m) => m.id)).toEqual(["s1a", "s1b"])
  })
})
