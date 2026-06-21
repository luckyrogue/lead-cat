import { describe, expect, it } from "vitest"

import { eventTypeKeys } from "./queries"

describe("eventTypeKeys", () => {
  it("scopes list key by org id", () => {
    expect(eventTypeKeys.list("org-123")).toEqual([
      "orgs",
      "org-123",
      "booking",
      "event-types",
    ])
  })

  it("uses distinct keys for different orgs", () => {
    expect(eventTypeKeys.list("org-a")).not.toEqual(eventTypeKeys.list("org-b"))
  })
})
