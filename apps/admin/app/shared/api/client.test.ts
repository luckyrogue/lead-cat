import { describe, expect, it } from "vitest"

import { prepareMutationCsrf } from "./client"

describe("prepareMutationCsrf", () => {
  it("skips GET requests", () => {
    expect(prepareMutationCsrf("get", null)).toEqual({})
  })

  it("returns header when csrf cookie is present", () => {
    expect(prepareMutationCsrf("post", "csrf-token")).toEqual({
      header: "csrf-token",
    })
  })

  it("warns in dev when csrf is missing", () => {
    expect(prepareMutationCsrf("delete", null, true)).toEqual({ warn: true })
  })

  it("throws in prod when csrf is missing", () => {
    expect(() => prepareMutationCsrf("patch", null, false)).toThrow(
      "missing_csrf_token"
    )
  })

  it("treats method case-insensitively", () => {
    expect(prepareMutationCsrf("PUT", "csrf-token")).toEqual({
      header: "csrf-token",
    })
  })
})
