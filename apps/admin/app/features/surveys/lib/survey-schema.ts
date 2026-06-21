import { z } from "zod"

import type { QuestionType, SurveyInput } from "~/entities/survey/types"

const questionSchema = z
  .object({
    prompt: z.string().min(1),
    type: z.enum(["single", "multi", "rating", "text"]),
    options: z.array(z.string().min(1)),
    rating_max: z.number().int(),
    required: z.boolean(),
  })
  .superRefine((q, ctx) => {
    if ((q.type === "single" || q.type === "multi") && q.options.length < 2) {
      ctx.addIssue({
        code: "custom",
        path: ["options"],
        message: "min_two_options",
      })
    }
    if (q.type === "rating" && (q.rating_max < 2 || q.rating_max > 10)) {
      ctx.addIssue({
        code: "custom",
        path: ["rating_max"],
        message: "rating_range",
      })
    }
  })

export const surveySchema = z.object({
  name: z.string().min(1),
  is_active: z.boolean(),
  questions: z.array(questionSchema).min(1),
})

export type SurveyForm = z.infer<typeof surveySchema>

export function emptyQuestion(type: QuestionType = "text") {
  return {
    prompt: "",
    type,
    options: type === "single" || type === "multi" ? ["", ""] : [],
    rating_max: 5,
    required: true,
  }
}

export function toSurveyInput(form: SurveyForm): SurveyInput {
  return form as SurveyInput
}
