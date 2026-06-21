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

  it("warns when csrf is missing on mutations", () => {
    expect(prepareMutationCsrf("delete", null)).toEqual({ warn: true })
    expect(prepareMutationCsrf("patch", null)).toEqual({ warn: true })
  })

  it("treats method case-insensitively", () => {
    expect(prepareMutationCsrf("PUT", "csrf-token")).toEqual({
      header: "csrf-token",
    })
  })
})
