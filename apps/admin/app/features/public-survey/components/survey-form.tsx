import { Button } from "@leadcat/ui"
import { useState } from "react"

import { useT } from "~/shared/i18n/context"
import { missingRequired } from "~/features/public-survey/lib/answers"
import {
  submitPublicSurvey,
  type AnswerValue,
  type PublicQuestion,
} from "~/features/public-survey/api"

export function SurveyForm({
  token,
  questions,
  onResult,
}: {
  token: string
  questions: PublicQuestion[]
  onResult: (r: "ok" | "completed" | "invalid") => void
}) {
  const t = useT()
  const [values, setValues] = useState<Record<string, AnswerValue>>({})
  const [missing, setMissing] = useState<string[]>([])
  const [submitting, setSubmitting] = useState(false)

  const set = (id: string, v: AnswerValue) =>
    setValues((p) => ({ ...p, [id]: v }))

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const m = missingRequired(questions, values)
    setMissing(m)
    if (m.length) return
    setSubmitting(true)
    const answers = questions
      .filter((q) => values[q.id] !== undefined && values[q.id] !== "")
      .map((q) => ({ question_id: q.id, value: values[q.id] }))
    try {
      onResult(await submitPublicSurvey(token, answers))
    } catch {
      onResult("invalid")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {questions.map((q) => (
        <div key={q.id} className="space-y-2">
          <p className="text-sm font-medium">
            {q.prompt}
            {q.required ? " *" : ""}
          </p>
          {q.type === "text" && (
            <textarea
              value={(values[q.id] as string) ?? ""}
              onChange={(e) => set(q.id, e.target.value)}
              rows={3}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
            />
          )}
          {q.type === "single" &&
            q.options.map((o) => (
              <label key={o} className="flex items-center gap-2 text-sm">
                <input
                  type="radio"
                  name={q.id}
                  checked={values[q.id] === o}
                  onChange={() => set(q.id, o)}
                />
                {o}
              </label>
            ))}
          {q.type === "multi" &&
            q.options.map((o) => {
              const arr = (values[q.id] as string[]) ?? []
              return (
                <label key={o} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={arr.includes(o)}
                    onChange={() =>
                      set(
                        q.id,
                        arr.includes(o)
                          ? arr.filter((x) => x !== o)
                          : [...arr, o]
                      )
                    }
                  />
                  {o}
                </label>
              )
            })}
          {q.type === "rating" && (
            <div className="flex gap-1.5">
              {Array.from({ length: q.rating_max }, (_, i) => i + 1).map(
                (n) => (
                  <Button
                    key={n}
                    type="button"
                    variant={values[q.id] === n ? "default" : "outline"}
                    size="icon"
                    onClick={() => set(q.id, n)}
                  >
                    {n}
                  </Button>
                )
              )}
            </div>
          )}
          {missing.includes(q.id) && (
            <p className="text-xs text-destructive">
              {t("publicSurvey.requiredError")}
            </p>
          )}
        </div>
      ))}
      <Button type="submit" className="w-full" disabled={submitting}>
        {submitting ? t("publicSurvey.submitting") : t("publicSurvey.submit")}
      </Button>
    </form>
  )
}
