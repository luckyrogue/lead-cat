import { describe, expect, it } from "vitest"

import { groupBySeries } from "./group-series"

type Row = { id: string; series_id?: string | null }

describe("groupBySeries", () => {
  it("groups recurring meetings by series_id", () => {
    const meetings: Row[] = [
      { id: "a", series_id: "s1" },
      { id: "b", series_id: "s1" },
      { id: "c" },
    ]
    const groups = groupBySeries(meetings)
    expect(groups).toHaveLength(2)
    if (groups[0]?.kind !== "series") {
      throw new Error("expected first group to be a series")
    }
    expect(groups[0].seriesId).toBe("s1")
    expect(groups[0].meetings).toHaveLength(2)
    expect(groups[1]).toEqual({ kind: "single", meeting: { id: "c" } })
  })
})
