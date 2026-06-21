import { describe, expect, it } from "vitest"

import { surveyKeys } from "./queries"

describe("surveyKeys", () => {
  it("scopes the list key by org id", () => {
    expect(surveyKeys.list("org-1")).toEqual(["orgs", "org-1", "surveys"])
  })
  it("distinguishes detail and responses keys", () => {
    expect(surveyKeys.detail("o", "s")).toEqual(["orgs", "o", "surveys", "s"])
    expect(surveyKeys.responses("o", "s", { status: "completed" })).toEqual([
      "orgs",
      "o",
      "surveys",
      "s",
      "responses",
      { status: "completed" },
    ])
  })
})
