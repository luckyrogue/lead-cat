export type AnswerValue = string | string[] | number

export type PublicQuestion = {
  id: string
  prompt: string
  type: "single" | "multi" | "rating" | "text"
  options: string[]
  rating_max: number
  required: boolean
}

export type PublicSurvey = {
  survey_name: string
  booker_name: string
  questions: PublicQuestion[]
}

const base = (import.meta.env.VITE_API_URL ?? "").replace(/\/$/, "")

export async function getPublicSurvey(
  token: string
): Promise<
  | { status: "ok"; survey: PublicSurvey }
  | { status: "not_found" }
  | { status: "completed" }
> {
  const res = await fetch(`${base}/api/survey/${encodeURIComponent(token)}`)
  if (res.ok)
    return { status: "ok", survey: (await res.json()) as PublicSurvey }
  if (res.status === 409) return { status: "completed" }
  return { status: "not_found" }
}

export async function submitPublicSurvey(
  token: string,
  answers: Array<{ question_id: string; value: AnswerValue }>
): Promise<"ok" | "completed" | "invalid"> {
  const res = await fetch(`${base}/api/survey/${encodeURIComponent(token)}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ answers }),
  })
  if (res.ok) return "ok"
  if (res.status === 409) return "completed"
  return "invalid"
}
