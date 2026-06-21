import type { AnswerValue, PublicQuestion } from "../api"

export function missingRequired(
  questions: PublicQuestion[],
  values: Record<string, AnswerValue | undefined>
): string[] {
  return questions
    .filter((q) => q.required && isEmpty(values[q.id]))
    .map((q) => q.id)
}

function isEmpty(v: AnswerValue | undefined): boolean {
  if (v === undefined || v === null) return true
  if (typeof v === "string") return v.trim() === ""
  if (Array.isArray(v)) return v.length === 0
  return false
}
