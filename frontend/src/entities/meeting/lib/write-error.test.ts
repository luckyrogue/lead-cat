import { describe, expect, it } from "vitest"
import { ApiError } from "@/shared/api/types"
import { writeErrorKey } from "./write-error"

describe("writeErrorKey", () => {
  it("maps meetings_not_configured → errNotConfigured", () => {
    expect(
      writeErrorKey(new ApiError(503, "x", "meetings_not_configured"))
    ).toBe("errNotConfigured")
  })
  it("maps meetings_recurring_unsupported → recurringSoon", () => {
    expect(
      writeErrorKey(new ApiError(400, "x", "meetings_recurring_unsupported"))
    ).toBe("recurringSoon")
  })
  it("maps code=forbidden → errNotYours", () => {
    expect(writeErrorKey(new ApiError(403, "x", "forbidden"))).toBe(
      "errNotYours"
    )
  })
  it("maps status=403 (no code) → errNotYours", () => {
    expect(writeErrorKey(new ApiError(403, "x"))).toBe("errNotYours")
  })
  it("maps status=404 → errNotYours", () => {
    expect(writeErrorKey(new ApiError(404, "x"))).toBe("errNotYours")
  })
  it("falls back to errGeneric on unknown ApiError", () => {
    expect(writeErrorKey(new ApiError(500, "x"))).toBe("errGeneric")
  })
  it("falls back to errGeneric on non-ApiError", () => {
    expect(writeErrorKey(new Error("nope"))).toBe("errGeneric")
    expect(writeErrorKey("string")).toBe("errGeneric")
    expect(writeErrorKey(null)).toBe("errGeneric")
  })
})
