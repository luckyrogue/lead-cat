import { describe, expect, it } from "vitest"

import { prepareMutationCsrf } from "./client"

describe("prepareMutationCsrf", () => {
  it("skips GET requests", () => {
    expect(prepareMutationCsrf("get", null, "/api/meetings")).toEqual({})
  })

  it("returns header when csrf cookie is present", () => {
    expect(prepareMutationCsrf("post", "csrf-token", "/api/meetings")).toEqual({
      header: "csrf-token",
    })
  })

  it("warns in dev when csrf is missing on a non-auth mutation", () => {
    expect(prepareMutationCsrf("delete", null, "/api/meetings", true)).toEqual({
      warn: true,
    })
  })

  it("throws in prod when csrf is missing on a non-auth mutation", () => {
    expect(() =>
      prepareMutationCsrf("patch", null, "/api/meetings", false)
    ).toThrow("missing_csrf_token")
  })

  it("allows auth-path mutations without csrf (login bootstrap)", () => {
    expect(
      prepareMutationCsrf("post", null, "/api/auth/web/magic/request", false)
    ).toEqual({})
  })

  it("treats method case-insensitively", () => {
    expect(prepareMutationCsrf("PUT", "csrf-token", "/api/meetings")).toEqual({
      header: "csrf-token",
    })
  })
})
