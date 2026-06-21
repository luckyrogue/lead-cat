import { describe, expect, it } from "vitest"

import { surveySchema, toSurveyInput } from "./survey-schema"

const base = {
  name: "S",
  is_active: true,
  questions: [
    {
      prompt: "Why?",
      type: "text",
      options: [],
      rating_max: 5,
      required: true,
    },
  ],
}

describe("surveySchema", () => {
  it("accepts a valid survey", () => {
    expect(surveySchema.safeParse(base).success).toBe(true)
  })
  it("rejects an empty name", () => {
    expect(surveySchema.safeParse({ ...base, name: "" }).success).toBe(false)
  })
  it("rejects zero questions", () => {
    expect(surveySchema.safeParse({ ...base, questions: [] }).success).toBe(
      false
    )
  })
  it("requires >=2 options for single", () => {
    const q = {
      prompt: "p",
      type: "single",
      options: ["a"],
      rating_max: 5,
      required: true,
    }
    expect(surveySchema.safeParse({ ...base, questions: [q] }).success).toBe(
      false
    )
  })
  it("requires rating_max in 2..10", () => {
    const q = {
      prompt: "p",
      type: "rating",
      options: [],
      rating_max: 1,
      required: true,
    }
    expect(surveySchema.safeParse({ ...base, questions: [q] }).success).toBe(
      false
    )
  })
})

describe("toSurveyInput", () => {
  it("passes through the validated shape", () => {
    const input = toSurveyInput(surveySchema.parse(base))
    expect(input.questions?.[0]?.prompt).toBe("Why?")
  })
})
