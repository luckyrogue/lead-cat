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

  it("surfaces the message for a 4xx ApiError that carries no code", () => {
    const err = new ApiError(400, "Bad request")
    expect(mapApiErrorMessage(t, err, "common.errors.generic")).toBe(
      "Bad request"
    )
  })

  it("never leaks a raw 5xx server message, using the fallback instead", () => {
    const err = new ApiError(500, "pq: relation does not exist", "DB_ERROR")
    expect(mapApiErrorMessage(t, err, "common.errors.generic")).toBe(
      "Something went wrong"
    )
  })

  it("still translates a known code even on a 5xx status", () => {
    const err = new ApiError(503, "rate limited", "RATE_LIMITED")
    expect(mapApiErrorMessage(t, err, "common.errors.generic")).toBe(
      "Too many requests"
    )
  })

  it("uses a plain Error's message", () => {
    expect(
      mapApiErrorMessage(t, new Error("boom"), "common.errors.generic")
    ).toBe("boom")
  })

  it("falls back when an Error has an empty message", () => {
    expect(mapApiErrorMessage(t, new Error(""), "common.errors.generic")).toBe(
      "Something went wrong"
    )
  })

  it("falls back to the translated fallback key for non-Error values", () => {
    expect(mapApiErrorMessage(t, null, "common.errors.generic")).toBe(
      "Something went wrong"
    )
    expect(mapApiErrorMessage(t, "oops", "common.errors.generic")).toBe(
      "Something went wrong"
    )
  })
})
