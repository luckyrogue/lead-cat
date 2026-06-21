import { describe, expect, it } from "vitest"

import { missingRequired } from "./answers"
import type { PublicQuestion } from "../api"

const q = (over: Partial<PublicQuestion>): PublicQuestion => ({
  id: "q",
  prompt: "p",
  type: "text",
  options: [],
  rating_max: 5,
  required: true,
  ...over,
})

describe("missingRequired", () => {
  it("flags an unanswered required text question", () => {
    expect(missingRequired([q({ id: "a" })], {})).toEqual(["a"])
  })
  it("accepts a filled required question", () => {
    expect(missingRequired([q({ id: "a" })], { a: "hi" })).toEqual([])
  })
  it("treats an empty multi-array as unanswered", () => {
    expect(
      missingRequired([q({ id: "a", type: "multi", options: ["x"] })], {
        a: [],
      })
    ).toEqual(["a"])
  })
  it("ignores optional questions", () => {
    expect(missingRequired([q({ id: "a", required: false })], {})).toEqual([])
  })
})
