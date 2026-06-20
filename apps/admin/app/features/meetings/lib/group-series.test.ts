import { describe, expect, it } from "vitest"
import { groupBySeries } from "./group-series"
import type { Meeting } from "~/entities/meeting/types"

const mk = (id: string, series_id?: string) =>
  ({ id, series_id }) as unknown as Meeting

describe("groupBySeries (admin)", () => {
  it("returns [] for empty input", () => {
    expect(groupBySeries([])).toEqual([])
  })
  it("groups by series in first-seen order with singles appended", () => {
    const result = groupBySeries([mk("s1a", "s1"), mk("solo"), mk("s1b", "s1")])
    expect(result.map((g) => g.kind)).toEqual(["series", "single"])
    const first = result[0]
    if (first.kind !== "series") {
      throw new Error("expected first group to be a series")
    }
    expect(first.meetings.map((m) => m.id)).toEqual(["s1a", "s1b"])
  })
})
