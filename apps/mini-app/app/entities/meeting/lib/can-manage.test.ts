import { describe, expect, it } from "vitest"

import { canManageMeeting } from "./can-manage"

describe("canManageMeeting", () => {
  const meeting = { organizer: "org@test.com" }

  it("allows organizer", () => {
    expect(
      canManageMeeting(meeting, { email: "org@test.com", role: "user" })
    ).toBe(true)
  })

  it("allows org owner", () => {
    expect(
      canManageMeeting(meeting, { email: "other@test.com", role: "owner" })
    ).toBe(true)
  })

  it("denies unrelated user", () => {
    expect(
      canManageMeeting(meeting, { email: "x@test.com", role: "user" })
    ).toBe(false)
  })
})
