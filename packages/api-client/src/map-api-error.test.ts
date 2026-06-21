import { describe, expect, it } from "vitest"

import { ApiError } from "./errors"
import { mapApiErrorMessage } from "./map-api-error"

// translate() returns the key itself when no translation exists.
const t = (key: string) => {
  if (key === "errors.codes.RATE_LIMITED") return "Too many requests"
  if (key === "common.errors.generic") return "Something went wrong"
  return key
}

describe("mapApiErrorMessage", () => {
  it("maps a known ApiError code to its translated message", () => {
    const err = new ApiError(429, "rate limited", "RATE_LIMITED")
    expect(mapApiErrorMessage(t, err, "common.errors.generic")).toBe(
      "Too many requests"
    )
  })

  it("uses the ApiError message when the code has no translation", () => {
    const err = new ApiError(400, "Email already taken", "EMAIL_TAKEN")
    expect(mapApiErrorMessage(t, err, "common.errors.generic")).toBe(
      "Email already taken"
    )
  })

  it("uses a plain Error's message", () => {
    expect(
      mapApiErrorMessage(t, new Error("boom"), "common.errors.generic")
    ).toBe("boom")
  })

  it("falls back to the translated fallback key for unknown values", () => {
    expect(mapApiErrorMessage(t, null, "common.errors.generic")).toBe(
      "Something went wrong"
    )
  })
})
