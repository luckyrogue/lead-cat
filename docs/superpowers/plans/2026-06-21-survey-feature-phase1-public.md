# Survey-on-decline Phase 1 — Public Survey Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the public, token-based survey page (`/survey/:token`) where a declined lead answers the operator's survey, and surface the survey CTA on the public booking form when a booking is declined.

**Architecture:** A public route in the admin SPA, like `book.$slug.tsx`, wrapped in `AuthLocaleShell` (public + localized + language switcher). Public endpoints are unauthenticated and use raw `fetch` (not the authenticated `~/shared/api/client`, which injects CSRF/`X-Org-Id` a public visitor doesn't have) — mirroring how `book.$slug.form.tsx` already calls `fetch("/api/book/:slug")`.

**Tech Stack:** React 19, react-router v7, raw `fetch`, `@leadcat/ui`, `useT()`/`useLocale()`. Depends on the **backend plan** (public endpoints live) and shares the admin i18n dictionaries.

## Global Constraints

- App root: `apps/admin`. Public routes use `AuthLocaleShell` for locale + language switching.
- Public calls use raw `fetch` to same-origin `/api/...`; no CSRF/org headers.
- Operator-authored question prompts/options are rendered as plain JSX text nodes (auto-escaped) — never `dangerouslySetInnerHTML`, never localized.
- Question types: `single` (radio), `multi` (checkboxes), `rating` (1…N buttons), `text` (textarea). Client-side `required` check before submit.
- All chrome copy via `t("...")` keys in `app/shared/i18n/dictionaries/{en,ru,kk}.ts` (typed parity enforced by typecheck).
- `pnpm --filter admin typecheck/lint/test` and `pnpm format` clean before each commit.

---

## File map

Create:
- `apps/admin/app/features/public-survey/api.ts`
- `apps/admin/app/features/public-survey/lib/answers.ts`
- `apps/admin/app/features/public-survey/lib/answers.test.ts`
- `apps/admin/app/features/public-survey/components/survey-form.tsx`
- `apps/admin/app/routes/survey.$token.tsx`

Modify:
- `apps/admin/app/routes/book.$slug.form.tsx` — show the survey CTA when the decline response carries `survey_token`.
- `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — add the `publicSurvey.*` keys (and the booking CTA key).

---

## Task 1: Public survey API + answer helpers

**Files:**
- Create: `apps/admin/app/features/public-survey/api.ts`
- Create: `apps/admin/app/features/public-survey/lib/answers.ts`
- Create: `apps/admin/app/features/public-survey/lib/answers.test.ts`

**Interfaces:**
- Produces:
  - Types `PublicQuestion { id; prompt; type; options: string[]; rating_max; required }`, `PublicSurvey { survey_name; booker_name; questions: PublicQuestion[] }`, `AnswerValue = string | string[] | number`.
  - `getPublicSurvey(token): Promise<{ status: "ok"; survey } | { status: "not_found" } | { status: "completed" }>`.
  - `submitPublicSurvey(token, answers): Promise<"ok" | "completed" | "invalid">`.
  - `missingRequired(questions, values): string[]` — ids of unanswered required questions.

- [ ] **Step 1: Write the failing `answers.test.ts`**

```ts
import { describe, expect, it } from "vitest"

import { missingRequired } from "./answers"
import type { PublicQuestion } from "../api"

const q = (over: Partial<PublicQuestion>): PublicQuestion => ({
  id: "q", prompt: "p", type: "text", options: [], rating_max: 5, required: true, ...over,
})

describe("missingRequired", () => {
  it("flags an unanswered required text question", () => {
    expect(missingRequired([q({ id: "a" })], {})).toEqual(["a"])
  })
  it("accepts a filled required question", () => {
    expect(missingRequired([q({ id: "a" })], { a: "hi" })).toEqual([])
  })
  it("treats an empty multi-array as unanswered", () => {
    expect(missingRequired([q({ id: "a", type: "multi", options: ["x"] })], { a: [] })).toEqual(["a"])
  })
  it("ignores optional questions", () => {
    expect(missingRequired([q({ id: "a", required: false })], {})).toEqual([])
  })
})
```

- [ ] **Step 2: Run to verify fail**

Run: `pnpm --filter admin exec vitest run app/features/public-survey/lib/answers.test.ts`
Expected: FAIL (cannot import).

- [ ] **Step 3: Implement `api.ts` then `answers.ts`**

`api.ts`:

```ts
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
  if (res.ok) return { status: "ok", survey: (await res.json()) as PublicSurvey }
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
```

`answers.ts`:

```ts
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
```

- [ ] **Step 4: Run to verify pass; commit**

Run: `pnpm --filter admin exec vitest run app/features/public-survey/lib/answers.test.ts`
Expected: PASS.

```bash
git add apps/admin/app/features/public-survey/api.ts apps/admin/app/features/public-survey/lib/
git commit -m "feat(public-survey): api client + required-answer helper"
```

---

## Task 2: Survey form component + route

**Files:**
- Create: `apps/admin/app/features/public-survey/components/survey-form.tsx`
- Create: `apps/admin/app/routes/survey.$token.tsx`

**Interfaces:**
- Consumes: `getPublicSurvey`, `submitPublicSurvey`, `missingRequired`, `@leadcat/ui`, `AuthLocaleShell`, `useT`.
- Produces: route `/survey/:token` rendering the page states.

- [ ] **Step 1: Implement `survey-form.tsx`**

A controlled form over `Record<questionId, AnswerValue>`:
- `single` → radio group over `options`.
- `multi` → checkbox group (toggle membership in a `string[]`).
- `rating` → buttons `1…rating_max` (selected highlighted).
- `text` → `textarea`.
On submit: compute `missingRequired`; if any, mark them and block; else build `answers` (`[{question_id, value}]`, skipping empty optional) and call `submitPublicSurvey`. Lift result to the page: `onResult("ok" | "completed" | "invalid")`. Localize chrome via `t()`; render `question.prompt` / `option` as plain text nodes.

```tsx
import { Button, Textarea } from "@leadcat/ui"
import { useState } from "react"

import { useT } from "~/shared/i18n/context"
import { missingRequired } from "~/features/public-survey/lib/answers"
import { submitPublicSurvey, type AnswerValue, type PublicQuestion } from "~/features/public-survey/api"

export function SurveyForm({ token, questions, onResult }: {
  token: string
  questions: PublicQuestion[]
  onResult: (r: "ok" | "completed" | "invalid") => void
}) {
  const t = useT()
  const [values, setValues] = useState<Record<string, AnswerValue>>({})
  const [missing, setMissing] = useState<string[]>([])
  const [submitting, setSubmitting] = useState(false)

  const set = (id: string, v: AnswerValue) => setValues((p) => ({ ...p, [id]: v }))

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const m = missingRequired(questions, values)
    setMissing(m)
    if (m.length) return
    setSubmitting(true)
    const answers = questions
      .filter((q) => values[q.id] !== undefined && values[q.id] !== "")
      .map((q) => ({ question_id: q.id, value: values[q.id] }))
    onResult(await submitPublicSurvey(token, answers))
    setSubmitting(false)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {questions.map((q) => (
        <div key={q.id} className="space-y-2">
          <p className="text-sm font-medium">
            {q.prompt}{q.required ? " *" : ""}
          </p>
          {q.type === "text" && (
            <Textarea value={(values[q.id] as string) ?? ""} onChange={(e) => set(q.id, e.target.value)} />
          )}
          {q.type === "single" && q.options.map((o) => (
            <label key={o} className="flex items-center gap-2 text-sm">
              <input type="radio" name={q.id} checked={values[q.id] === o} onChange={() => set(q.id, o)} />
              {o}
            </label>
          ))}
          {q.type === "multi" && q.options.map((o) => {
            const arr = (values[q.id] as string[]) ?? []
            return (
              <label key={o} className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={arr.includes(o)}
                  onChange={() => set(q.id, arr.includes(o) ? arr.filter((x) => x !== o) : [...arr, o])} />
                {o}
              </label>
            )
          })}
          {q.type === "rating" && (
            <div className="flex gap-1.5">
              {Array.from({ length: q.rating_max }, (_, i) => i + 1).map((n) => (
                <Button key={n} type="button" variant={values[q.id] === n ? "default" : "outline"}
                  size="icon" onClick={() => set(q.id, n)}>{n}</Button>
              ))}
            </div>
          )}
          {missing.includes(q.id) && <p className="text-destructive text-xs">{t("publicSurvey.requiredError")}</p>}
        </div>
      ))}
      <Button type="submit" className="w-full" disabled={submitting}>
        {submitting ? t("publicSurvey.submitting") : t("publicSurvey.submit")}
      </Button>
    </form>
  )
}
```

- [ ] **Step 2: Implement `survey.$token.tsx`**

```tsx
import { Card, CardContent, CardHeader, CardTitle } from "@leadcat/ui"
import { useEffect, useState } from "react"
import { useParams } from "react-router"

import { AuthLocaleShell } from "~/components/auth-locale-shell"
import { useT } from "~/shared/i18n/context"
import { getPublicSurvey, type PublicSurvey } from "~/features/public-survey/api"
import { SurveyForm } from "~/features/public-survey/components/survey-form"

type State =
  | { kind: "loading" }
  | { kind: "form"; survey: PublicSurvey }
  | { kind: "unavailable" }
  | { kind: "done" }

export default function Route() {
  return (
    <AuthLocaleShell>
      <SurveyScreen />
    </AuthLocaleShell>
  )
}

function SurveyScreen() {
  const t = useT()
  const { token = "" } = useParams()
  const [state, setState] = useState<State>({ kind: "loading" })

  useEffect(() => {
    let active = true
    getPublicSurvey(token).then((r) => {
      if (!active) return
      if (r.status === "ok") setState({ kind: "form", survey: r.survey })
      else if (r.status === "completed") setState({ kind: "done" })
      else setState({ kind: "unavailable" })
    })
    return () => { active = false }
  }, [token])

  return (
    <div className="mx-auto flex min-h-svh max-w-md items-center px-4 py-10">
      <Card className="w-full">
        {state.kind === "loading" && <CardContent className="py-10 text-center">{t("common.loading")}</CardContent>}
        {state.kind === "unavailable" && (
          <CardContent className="py-10 text-center text-muted-foreground">{t("publicSurvey.unavailable")}</CardContent>
        )}
        {state.kind === "done" && (
          <CardContent className="py-10 text-center">{t("publicSurvey.thanks")}</CardContent>
        )}
        {state.kind === "form" && (
          <>
            <CardHeader><CardTitle className="text-xl">{state.survey.survey_name}</CardTitle></CardHeader>
            <CardContent>
              <SurveyForm token={token} questions={state.survey.questions}
                onResult={(r) => setState(r === "ok" || r === "completed" ? { kind: "done" } : { kind: "form", survey: state.survey })} />
            </CardContent>
          </>
        )}
      </Card>
    </div>
  )
}
```

- [ ] **Step 3: Typecheck + lint (after i18n keys from Task 4)**

Run: `pnpm --filter admin typecheck && pnpm --filter admin lint`
Expected: clean once Task 4 keys exist.

- [ ] **Step 4: Commit**

```bash
git add apps/admin/app/features/public-survey/components/survey-form.tsx \
        apps/admin/app/routes/survey.\$token.tsx
git commit -m "feat(public-survey): /survey/:token page with all question types + states"
```

---

## Task 3: Booking-form CTA on decline

**Files:**
- Modify: `apps/admin/app/routes/book.$slug.form.tsx`

**Interfaces:**
- Consumes: the decline response body now carries `survey_token` (from the backend plan).
- Produces: a CTA linking to `/survey/:token` shown under the decline message.

- [ ] **Step 1: Capture `survey_token` from the decline response**

In `book.$slug.form.tsx`, the submit handler branches on `res.status`. For the 409 and 400 branches, parse the JSON body and store any `survey_token`:

```tsx
// add to FormState union:
//   | { status: "conflict"; surveyToken?: string }
//   | { status: "badInput"; surveyToken?: string }

if (res.status === 409) {
  const body = await res.json().catch(() => ({}))
  setFormState({ status: "conflict", surveyToken: body.survey_token })
  onConflict()
  return
}
if (res.status === 400) {
  const body = await res.json().catch(() => ({}))
  setFormState({ status: "badInput", surveyToken: body.survey_token })
  return
}
```

- [ ] **Step 2: Render the CTA**

Where the conflict/badInput messages render, add (when `formState.surveyToken` is set):

```tsx
{(formState.status === "conflict" || formState.status === "badInput") && formState.surveyToken ? (
  <a
    href={`/survey/${formState.surveyToken}`}
    className="bg-primary text-primary-foreground inline-flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-opacity hover:opacity-90"
  >
    {t("publicBooking.surveyCta")}
  </a>
) : null}
```

- [ ] **Step 3: Typecheck + lint**

Run: `pnpm --filter admin typecheck && pnpm --filter admin lint`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add apps/admin/app/routes/book.\$slug.form.tsx
git commit -m "feat(public-survey): booking-decline CTA links to the survey"
```

---

## Task 4: i18n keys (en/ru/kk)

**Files:**
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts`

> Implement before Tasks 2–3 typecheck. Keys must be identical across all three locales.

- [ ] **Step 1: Add keys to `en.ts`**

Add a `publicSurvey` namespace and one `publicBooking.surveyCta` key:

```ts
  publicSurvey: {
    unavailable: "This survey is no longer available.",
    thanks: "Thank you! Your answers were submitted.",
    submit: "Submit",
    submitting: "Submitting…",
    requiredError: "This question is required.",
  },
  // inside the existing publicBooking namespace:
  //   surveyCta: "Couldn't find a good time? Take a short survey →",
```

- [ ] **Step 2: Mirror into `ru.ts` and `kk.ts`**

Russian: `unavailable: "Этот опрос больше недоступен."`, `thanks: "Спасибо! Ваши ответы отправлены."`, `submit: "Отправить"`, `submitting: "Отправка…"`, `requiredError: "Этот вопрос обязателен."`, `publicBooking.surveyCta: "Не нашли удобное время? Пройдите короткий опрос →"`. Kazakh: equivalent translations. Keep keys identical.

- [ ] **Step 3: Verify parity + format**

Run: `pnpm --filter admin typecheck && pnpm format`
Expected: green.

- [ ] **Step 4: Commit**

```bash
git add apps/admin/app/shared/i18n/dictionaries/
git commit -m "feat(public-survey): i18n strings + booking CTA (en/ru/kk)"
```

---

## Task 5: Full verification

- [ ] **Step 1: Run the admin suite + build**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin test && pnpm --filter admin build
```
Expected: all green (build proves the public route compiles in the SPA bundle).

- [ ] **Step 2: Manual smoke (recommended)**

With the backend running and a survey assigned to a booking event-type, force a decline (book a taken slot), follow the CTA, submit the survey, and confirm the "thanks" state + a `completed` row in the admin responses table.

---

## Self-review notes (addressed)

- **Spec coverage:** public page on its own `/survey/:token` route (Task 2), 4 question types rendered (Task 2), page states loading/form/unavailable/done incl. 409 already-completed (Tasks 1–2), booking CTA on decline reading `survey_token` (Task 3), operator content not localized (rendered as text), chrome localized en/ru/kk (Task 4).
- **Type consistency:** `PublicQuestion`/`PublicSurvey`/`AnswerValue` defined in `api.ts` (Task 1) and consumed by `answers.ts`, `survey-form.tsx`, and the route; `missingRequired` signature fixed in Task 1.
- **Public-vs-authenticated:** public endpoints use raw `fetch` (no CSRF/org), matching `book.$slug.form.tsx`; explicitly not the `~/shared/api/client`.
- **Ordering dependency:** Task 4 (i18n) before Tasks 2–3 typecheck — noted.
