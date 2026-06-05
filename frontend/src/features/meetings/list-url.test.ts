import { describe, expect, it } from "vitest"
import {
  buildMeetingsSearchParams,
  parseMeetingsScopeFilter,
} from "@/features/meetings/list-url"
import {
  filterToScope,
  scopeToFilter,
} from "@/features/meetings/queries"

describe("meetings list URL helpers", () => {
  it("maps scope query param to filter", () => {
    expect(parseMeetingsScopeFilter(new URLSearchParams("scope=past"))).toBe(
      "past"
    )
    expect(parseMeetingsScopeFilter(new URLSearchParams("scope=all"))).toBe(
      "all"
    )
    expect(parseMeetingsScopeFilter(new URLSearchParams())).toBe("up")
  })

  it("serializes filter back to scope", () => {
    expect(filterToScope("up")).toBe("upcoming")
    expect(filterToScope("past")).toBe("past")
    expect(filterToScope("all")).toBe("all")
    expect(scopeToFilter("upcoming")).toBe("up")
  })

  it("builds search params with scope filter", () => {
    const params = buildMeetingsSearchParams({
      q: "demo",
      page: 2,
      filter: "past",
    })
    expect(params.get("q")).toBe("demo")
    expect(params.get("page")).toBe("2")
    expect(params.get("scope")).toBe("past")
  })
})
