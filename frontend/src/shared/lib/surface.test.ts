import { describe, it, expect } from "vitest"
import { detectSurface } from "./surface"

describe("detectSurface", () => {
  it("telegram when initData present", () => { expect(detectSurface({ initData: "x" })).toBe("telegram") })
  it("web otherwise", () => {
    expect(detectSurface(undefined)).toBe("web")
    expect(detectSurface(null)).toBe("web")
    expect(detectSurface({ initData: "" })).toBe("web")
    expect(detectSurface({})).toBe("web")
  })
})
