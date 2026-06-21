# Survey-on-decline Phase 1 — Admin UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the admin (operator) UI for surveys: a survey library, a builder with a question editor, a responses table with CSV export, and assignment of a survey to a booking event-type.

**Architecture:** Feature-Sliced Design, mirroring the existing booking feature. A new `entities/survey` layer (typed API + react-query hooks) and a new `features/surveys` slice (pages + builder dialog + question editor). The admin axios client (`~/shared/api/client`) already attaches `X-Org-Id` and CSRF automatically — survey calls need no org plumbing.

**Tech Stack:** React 19, react-router v7 (file routes), @tanstack/react-query, react-hook-form + zod, `@leadcat/ui` components, `@leadcat/api-client` generated types, vitest. Depends on the **backend plan** being implemented and OpenAPI regenerated (survey types exist in `@leadcat/api-client`).

## Global Constraints

- App root: `apps/admin`. FSD layers: `app` → `features` → `entities` → `shared`; never import upward.
- All user-facing copy goes through `useT()` / `t("...")`; add keys to `app/shared/i18n/dictionaries/{en,ru,kk}.ts`. The dictionaries are typed — a key missing in any locale fails `typecheck`.
- Operator-authored survey content (prompts/options) is never localized and never rendered via `dangerouslySetInnerHTML` — plain JSX text nodes only.
- Question types: `single | multi | rating | text`. Builder zod validation mirrors the backend: `single`/`multi` need ≥2 options; `rating` `rating_max` 2..10; `text` no options.
- Run `pnpm --filter admin typecheck`, `pnpm --filter admin lint`, `pnpm --filter admin test`, and `pnpm format` clean before each commit. Prettier uses `apps/admin/config/prettier.config.mjs`.

---

## File map

Create:
- `apps/admin/app/entities/survey/types.ts`
- `apps/admin/app/entities/survey/api.ts`
- `apps/admin/app/entities/survey/queries.ts`
- `apps/admin/app/entities/survey/queries.test.ts`
- `apps/admin/app/features/surveys/lib/survey-schema.ts`
- `apps/admin/app/features/surveys/lib/survey-schema.test.ts`
- `apps/admin/app/features/surveys/components/question-editor.tsx`
- `apps/admin/app/features/surveys/components/survey-dialog.tsx`
- `apps/admin/app/features/surveys/pages/surveys-page.tsx`
- `apps/admin/app/features/surveys/pages/responses-page.tsx`
- `apps/admin/app/routes/_app.surveys._index.tsx`
- `apps/admin/app/routes/_app.surveys.$id.responses.tsx`

Modify:
- `apps/admin/app/components/app-sidebar.tsx` — add the "Surveys" nav item.
- `apps/admin/app/features/booking/components/event-type-dialog.tsx` — add the survey assignment select.
- `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — add the `surveys.*` and `nav.surveys` keys.

---

## Task 1: Survey entity (types, api, queries)

**Files:**
- Create: `apps/admin/app/entities/survey/types.ts`
- Create: `apps/admin/app/entities/survey/api.ts`
- Create: `apps/admin/app/entities/survey/queries.ts`
- Create: `apps/admin/app/entities/survey/queries.test.ts`

**Interfaces:**
- Produces:
  - Types `Survey`, `SurveyQuestion`, `SurveyResponse`, `SurveyInput`, `ResponseFilter` (re-exported/derived from `@leadcat/api-client`; if the generated names differ, alias them here).
  - API fns: `listSurveys()`, `getSurvey(id)`, `createSurvey(input)`, `updateSurvey(id, input)`, `deleteSurvey(id)`, `listResponses(id, filter)`, `responsesCsvUrl(id, filter)`.
  - `surveyKeys.list(orgId)`, `surveyKeys.detail(orgId, id)`, `surveyKeys.responses(orgId, id, filter)`.
  - Hooks: `useSurveys`, `useSurvey`, `useCreateSurvey`, `useUpdateSurvey`, `useDeleteSurvey`, `useResponses`.

- [ ] **Step 1: Write `types.ts`**

```ts
import type {
  Survey as ApiSurvey,
  SurveyQuestion as ApiSurveyQuestion,
  SurveyResponse as ApiSurveyResponse,
  SurveyInput as ApiSurveyInput,
} from "@leadcat/api-client"

export type Survey = ApiSurvey
export type SurveyQuestion = ApiSurveyQuestion
export type SurveyResponse = ApiSurveyResponse
export type SurveyInput = ApiSurveyInput

export type QuestionType = "single" | "multi" | "rating" | "text"

export type ResponseFilter = {
  status?: "sent" | "completed"
  reason?: "slot_taken" | "invalid_booking"
  from?: string
  to?: string
}
```

> If the OpenAPI generator did not emit a named `SurveyInput`, define the input shape locally:
> `export type SurveyInput = { name: string; is_active: boolean; questions: Array<{ prompt: string; type: QuestionType; options: string[]; rating_max: number; required: boolean }> }`.

- [ ] **Step 2: Write `api.ts`**

```ts
import { api } from "~/shared/api/client"
import type {
  ResponseFilter,
  Survey,
  SurveyInput,
  SurveyResponse,
} from "~/entities/survey/types"

type SurveysResponse = { surveys: Survey[] }
type ResponsesResponse = { survey: Survey; responses: SurveyResponse[] }

export async function listSurveys(): Promise<Survey[]> {
  const { data } = await api.get<SurveysResponse>("/api/surveys")
  return data.surveys ?? []
}

export async function getSurvey(id: string): Promise<Survey> {
  const { data } = await api.get<Survey>(`/api/surveys/${id}`)
  return data
}

export async function createSurvey(input: SurveyInput): Promise<Survey> {
  const { data } = await api.post<Survey>("/api/surveys", input)
  return data
}

export async function updateSurvey(
  id: string,
  input: SurveyInput
): Promise<Survey> {
  const { data } = await api.patch<Survey>(`/api/surveys/${id}`, input)
  return data
}

export async function deleteSurvey(id: string): Promise<void> {
  await api.delete(`/api/surveys/${id}`)
}

function filterQuery(filter: ResponseFilter): string {
  const params = new URLSearchParams()
  if (filter.status) params.set("status", filter.status)
  if (filter.reason) params.set("reason", filter.reason)
  if (filter.from) params.set("from", filter.from)
  if (filter.to) params.set("to", filter.to)
  const q = params.toString()
  return q ? `?${q}` : ""
}

export async function listResponses(
  id: string,
  filter: ResponseFilter = {}
): Promise<ResponsesResponse> {
  const { data } = await api.get<ResponsesResponse>(
    `/api/surveys/${id}/responses${filterQuery(filter)}`
  )
  return data
}

export function responsesCsvPath(id: string, filter: ResponseFilter = {}): string {
  return `/api/surveys/${id}/responses.csv${filterQuery(filter)}`
}
```

- [ ] **Step 3: Write the failing query-keys test**

```ts
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
```

- [ ] **Step 4: Run test to verify it fails**

Run: `pnpm --filter admin test -- survey` (or `pnpm --filter admin exec vitest run app/entities/survey/queries.test.ts`)
Expected: FAIL (cannot import `surveyKeys`).

- [ ] **Step 5: Write `queries.ts`**

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  createSurvey,
  deleteSurvey,
  getSurvey,
  listResponses,
  listSurveys,
  updateSurvey,
} from "~/entities/survey/api"
import type { ResponseFilter, SurveyInput } from "~/entities/survey/types"

export const surveyKeys = {
  list: (orgId: string) => ["orgs", orgId, "surveys"] as const,
  detail: (orgId: string, id: string) =>
    ["orgs", orgId, "surveys", id] as const,
  responses: (orgId: string, id: string, filter: ResponseFilter) =>
    ["orgs", orgId, "surveys", id, "responses", filter] as const,
}

export function useSurveys(orgId: string | null) {
  return useQuery({
    queryKey: surveyKeys.list(orgId ?? ""),
    queryFn: listSurveys,
    enabled: Boolean(orgId),
  })
}

export function useSurvey(orgId: string | null, id: string) {
  return useQuery({
    queryKey: surveyKeys.detail(orgId ?? "", id),
    queryFn: () => getSurvey(id),
    enabled: Boolean(orgId && id),
  })
}

export function useResponses(
  orgId: string | null,
  id: string,
  filter: ResponseFilter
) {
  return useQuery({
    queryKey: surveyKeys.responses(orgId ?? "", id, filter),
    queryFn: () => listResponses(id, filter),
    enabled: Boolean(orgId && id),
  })
}

export function useCreateSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SurveyInput) => createSurvey(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}

export function useUpdateSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (args: { id: string; input: SurveyInput }) =>
      updateSurvey(args.id, args.input),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}

export function useDeleteSurvey(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSurvey(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: surveyKeys.list(orgId) }),
  })
}
```

- [ ] **Step 6: Run test to verify it passes; commit**

Run: `pnpm --filter admin exec vitest run app/entities/survey/queries.test.ts`
Expected: PASS.

```bash
git add apps/admin/app/entities/survey/
git commit -m "feat(surveys-admin): survey entity — types, api, query hooks"
```

---

## Task 2: Builder zod schema (validation, pure + tested)

**Files:**
- Create: `apps/admin/app/features/surveys/lib/survey-schema.ts`
- Create: `apps/admin/app/features/surveys/lib/survey-schema.test.ts`

**Interfaces:**
- Produces: `surveySchema` (zod), `type SurveyForm = z.infer<typeof surveySchema>`, `emptyQuestion(type)` factory, `toSurveyInput(form): SurveyInput`.

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, it } from "vitest"

import { surveySchema, toSurveyInput } from "./survey-schema"

const base = {
  name: "S",
  is_active: true,
  questions: [
    { prompt: "Why?", type: "text", options: [], rating_max: 5, required: true },
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
    expect(surveySchema.safeParse({ ...base, questions: [] }).success).toBe(false)
  })
  it("requires >=2 options for single", () => {
    const q = { prompt: "p", type: "single", options: ["a"], rating_max: 5, required: true }
    expect(surveySchema.safeParse({ ...base, questions: [q] }).success).toBe(false)
  })
  it("requires rating_max in 2..10", () => {
    const q = { prompt: "p", type: "rating", options: [], rating_max: 1, required: true }
    expect(surveySchema.safeParse({ ...base, questions: [q] }).success).toBe(false)
  })
})

describe("toSurveyInput", () => {
  it("passes through the validated shape", () => {
    expect(toSurveyInput(surveySchema.parse(base)).questions[0].prompt).toBe("Why?")
  })
})
```

- [ ] **Step 2: Run to verify fail**

Run: `pnpm --filter admin exec vitest run app/features/surveys/lib/survey-schema.test.ts`
Expected: FAIL (cannot import).

- [ ] **Step 3: Implement `survey-schema.ts`**

```ts
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
      ctx.addIssue({ code: "custom", path: ["options"], message: "min_two_options" })
    }
    if (q.type === "rating" && (q.rating_max < 2 || q.rating_max > 10)) {
      ctx.addIssue({ code: "custom", path: ["rating_max"], message: "rating_range" })
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
```

- [ ] **Step 4: Run to verify pass; commit**

Run: `pnpm --filter admin exec vitest run app/features/surveys/lib/survey-schema.test.ts`
Expected: PASS.

```bash
git add apps/admin/app/features/surveys/lib/
git commit -m "feat(surveys-admin): builder zod schema mirroring backend validation"
```

---

## Task 3: Question editor + survey dialog

**Files:**
- Create: `apps/admin/app/features/surveys/components/question-editor.tsx`
- Create: `apps/admin/app/features/surveys/components/survey-dialog.tsx`

**Interfaces:**
- Consumes: `surveySchema`, `emptyQuestion`, `toSurveyInput`, `useCreateSurvey`/`useUpdateSurvey`, `@leadcat/ui` (`Dialog`, `Input`, `Switch`, `Select`, `Button`, `Label`), `useFieldArray`/`useForm`.
- Produces: `<QuestionEditor control={...} name="questions" />` and `<SurveyDialog open onOpenChange survey?={Survey} orgId />`.

- [ ] **Step 1: Implement `question-editor.tsx`**

Render the `questions` field array: per question a card with prompt input, type `Select` (single/multi/rating/text via localized labels), a `required` `Switch`, conditional options editor (add/remove rows) for single/multi, a `rating_max` `Select` (2–10) for rating, and ↑/↓/remove buttons. Use `useFieldArray({ control, name: "questions" })` with `move`, `remove`, `append`. Follow the field/label/error pattern in `apps/admin/app/features/booking/components/event-type-dialog-field.tsx`. Localize all chrome via `useT()`; render the operator's prompt/option *inputs* as plain controlled fields (their content is not localized).

```tsx
import { Button, Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Switch } from "@leadcat/ui"
import { Plus, Trash2, ChevronUp, ChevronDown } from "lucide-react"
import { Controller, useFieldArray, type Control } from "react-hook-form"

import { useT } from "~/shared/i18n/context"
import { emptyQuestion } from "~/features/surveys/lib/survey-schema"
import type { SurveyForm } from "~/features/surveys/lib/survey-schema"

const TYPES = ["single", "multi", "rating", "text"] as const

export function QuestionEditor({ control }: { control: Control<SurveyForm> }) {
  const t = useT()
  const { fields, append, remove, move } = useFieldArray({ control, name: "questions" })
  return (
    <div className="space-y-3">
      {fields.map((field, i) => (
        <div key={field.id} className="rounded-lg border p-3 space-y-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">{t("surveys.question")} {i + 1}</span>
            <div className="flex gap-1">
              <Button type="button" variant="ghost" size="icon" disabled={i === 0} onClick={() => move(i, i - 1)}><ChevronUp className="size-4" /></Button>
              <Button type="button" variant="ghost" size="icon" disabled={i === fields.length - 1} onClick={() => move(i, i + 1)}><ChevronDown className="size-4" /></Button>
              <Button type="button" variant="ghost" size="icon" onClick={() => remove(i)}><Trash2 className="size-4" /></Button>
            </div>
          </div>

          <Controller control={control} name={`questions.${i}.prompt`} render={({ field }) => (
            <Input {...field} placeholder={t("surveys.questionPrompt")} />
          )} />

          <div className="grid grid-cols-2 gap-2">
            <Controller control={control} name={`questions.${i}.type`} render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {TYPES.map((ty) => <SelectItem key={ty} value={ty}>{t(`surveys.type.${ty}`)}</SelectItem>)}
                </SelectContent>
              </Select>
            )} />
            <Controller control={control} name={`questions.${i}.required`} render={({ field }) => (
              <label className="flex items-center gap-2 text-sm">
                <Switch checked={field.value} onCheckedChange={field.onChange} />
                {t("surveys.required")}
              </label>
            )} />
          </div>

          <Controller control={control} name={`questions.${i}.type`} render={({ field: typeField }) => (
            <QuestionExtras control={control} index={i} type={typeField.value} />
          )} />
        </div>
      ))}
      <Button type="button" variant="outline" onClick={() => append(emptyQuestion("text"))}>
        <Plus className="size-4" /> {t("surveys.addQuestion")}
      </Button>
    </div>
  )
}

function QuestionExtras({ control, index, type }: { control: Control<SurveyForm>; index: number; type: string }) {
  const t = useT()
  const { fields, append, remove } = useFieldArray({ control, name: `questions.${index}.options` as const })
  if (type === "single" || type === "multi") {
    return (
      <div className="space-y-1.5">
        <Label className="text-xs">{t("surveys.options")}</Label>
        {fields.map((f, oi) => (
          <div key={f.id} className="flex gap-2">
            <Controller control={control} name={`questions.${index}.options.${oi}`} render={({ field }) => (
              <Input {...field} placeholder={`${t("surveys.option")} ${oi + 1}`} />
            )} />
            <Button type="button" variant="ghost" size="icon" onClick={() => remove(oi)}><Trash2 className="size-4" /></Button>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={() => append("")}>{t("surveys.addOption")}</Button>
      </div>
    )
  }
  if (type === "rating") {
    return (
      <Controller control={control} name={`questions.${index}.rating_max`} render={({ field }) => (
        <Select value={String(field.value)} onValueChange={(v) => field.onChange(Number(v))}>
          <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
          <SelectContent>
            {[2,3,4,5,6,7,8,9,10].map((n) => <SelectItem key={n} value={String(n)}>{n}</SelectItem>)}
          </SelectContent>
        </Select>
      )} />
    )
  }
  return null
}
```

- [ ] **Step 2: Implement `survey-dialog.tsx`**

Wrap a `Dialog` with `useForm({ resolver: zodResolver(surveySchema), defaultValues })`; render name `Input`, `is_active` `Switch`, `<QuestionEditor control={control} />`, and submit via `useCreateSurvey`/`useUpdateSurvey` (choose by whether `survey` prop is set). On success: toast + close. Mirror the structure of `event-type-dialog.tsx` exactly (header, form, footer buttons, error toasts via `toastApiError`). Default values for create: `{ name: "", is_active: true, questions: [emptyQuestion("text")] }`; for edit, map the existing `survey` (including its questions) into the form shape.

- [ ] **Step 3: Typecheck + lint**

Run: `pnpm --filter admin typecheck && pnpm --filter admin lint`
Expected: clean (keys referenced here are added in Task 6; if typecheck fails only on missing `t` keys, proceed — they land in Task 6, but to keep this task green, add the keys in Task 6 *before* committing the slice, or run Task 6 first).

> Ordering note: run **Task 6 (i18n keys) before** Task 3's commit, or fold the key additions into this commit, so `typecheck` is green. The plan lists Task 6 separately for clarity, but its keys are a prerequisite for Tasks 3–5 to typecheck.

- [ ] **Step 4: Commit**

```bash
git add apps/admin/app/features/surveys/components/
git commit -m "feat(surveys-admin): question editor + survey builder dialog"
```

---

## Task 4: Surveys list page + route + nav

**Files:**
- Create: `apps/admin/app/features/surveys/pages/surveys-page.tsx`
- Create: `apps/admin/app/routes/_app.surveys._index.tsx`
- Modify: `apps/admin/app/components/app-sidebar.tsx`

**Interfaces:**
- Consumes: `useSurveys`, `useDeleteSurvey`, `SurveyDialog`, `useActiveOrgId`.
- Produces: route at `/surveys`; nav item `nav.surveys`.

- [ ] **Step 1: Implement `surveys-page.tsx`**

List org surveys (cards/rows): name, question count, active badge, "N questions", buttons Edit (opens `SurveyDialog`), Responses (`<Link to={\`/surveys/${id}/responses\`}>`), Delete (calls `useDeleteSurvey`; on 409 `survey_has_responses` show toast "deactivate instead"). A "Create survey" button opens `SurveyDialog` with no `survey`. Use `getActiveOrgId()`/`useActiveOrgId` for `orgId`. Follow the layout of `features/booking/pages/booking-page.tsx`.

- [ ] **Step 2: Create the route file**

```tsx
import { SurveysPage } from "~/features/surveys/pages/surveys-page"

export default function Route() {
  return <SurveysPage />
}
```

- [ ] **Step 3: Add the nav item**

In `apps/admin/app/components/app-sidebar.tsx`, import an icon (e.g. `ClipboardList` from `lucide-react`) and add to `navItems` after the booking entry:

```tsx
  { href: "/surveys", labelKey: "nav.surveys", icon: ClipboardList },
```

- [ ] **Step 4: Typecheck + manual smoke**

Run: `pnpm --filter admin typecheck`
Expected: clean (with Task 6 keys present). Manually: nav shows "Surveys", page lists/creates.

- [ ] **Step 5: Commit**

```bash
git add apps/admin/app/features/surveys/pages/surveys-page.tsx \
        apps/admin/app/routes/_app.surveys._index.tsx \
        apps/admin/app/components/app-sidebar.tsx
git commit -m "feat(surveys-admin): surveys list page, route, nav item"
```

---

## Task 5: Responses page (table + filters + CSV) + assignment select

**Files:**
- Create: `apps/admin/app/features/surveys/pages/responses-page.tsx`
- Create: `apps/admin/app/routes/_app.surveys.$id.responses.tsx`
- Modify: `apps/admin/app/features/booking/components/event-type-dialog.tsx`

**Interfaces:**
- Consumes: `useResponses`, `responsesCsvPath`, `useParams`, `useSurveys` (for the assignment select), the api `baseUrl`.
- Produces: route `/surveys/:id/responses`; a "Survey on decline" select inside the event-type dialog.

- [ ] **Step 1: Implement `responses-page.tsx`**

Read `id` from `useParams()`. Local filter state (`status`, `reason`, `from`, `to`) → `useResponses(orgId, id, filter)`. Render a table: date, name, email, service (decline reason), status badge, and an expandable answers cell (map `response.answers` → `prompt: value`, joining `multi` arrays with "; "). A filter bar (selects + date inputs). An "Export CSV" button that triggers a download of `${baseUrl}${responsesCsvPath(id, filter)}` (use an `<a href download>` or `window.open`; cookies are sent automatically same-origin). Follow `features/meetings` table styling.

- [ ] **Step 2: Create the route file**

```tsx
import { ResponsesPage } from "~/features/surveys/pages/responses-page"

export default function Route() {
  return <ResponsesPage />
}
```

- [ ] **Step 3: Add the assignment select to the event-type dialog**

In `event-type-dialog.tsx`: load `useSurveys(orgId)`, add a `Controller`-driven `Select` labeled `surveys.assignLabel` whose value is the event-type's `survey_id` (`""` = `surveys.assignNone`, otherwise a survey id). Include the field in the dialog's form and send it as `survey_id` in the existing event-type create/update payload (the backend PATCH accepts `survey_id`). Only list `is_active` surveys plus the currently-assigned one.

- [ ] **Step 4: Typecheck + lint + test**

Run: `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin test`
Expected: clean, tests green.

- [ ] **Step 5: Commit**

```bash
git add apps/admin/app/features/surveys/pages/responses-page.tsx \
        apps/admin/app/routes/_app.surveys.\$id.responses.tsx \
        apps/admin/app/features/booking/components/event-type-dialog.tsx
git commit -m "feat(surveys-admin): responses table + CSV export + event-type assignment"
```

---

## Task 6: i18n keys (en/ru/kk)

**Files:**
- Modify: `apps/admin/app/shared/i18n/dictionaries/en.ts`
- Modify: `apps/admin/app/shared/i18n/dictionaries/ru.ts`
- Modify: `apps/admin/app/shared/i18n/dictionaries/kk.ts`

> Implement this task **first or alongside Task 3** so Tasks 3–5 typecheck (the dictionaries are typed and shared).

**Interfaces:** adds `nav.surveys` and a `surveys` namespace used across the slice.

- [ ] **Step 1: Add `nav.surveys` and the `surveys` block to `en.ts`**

Add `surveys: "Surveys"` under `nav`, and a top-level `surveys` namespace:

```ts
  surveys: {
    title: "Surveys",
    create: "Create survey",
    name: "Survey name",
    active: "Active",
    question: "Question",
    questionPrompt: "Question text",
    addQuestion: "Add question",
    required: "Required",
    options: "Options",
    option: "Option",
    addOption: "Add option",
    type: { single: "Single choice", multi: "Multiple choice", rating: "Rating", text: "Free text" },
    responses: "Responses",
    responsesTitle: "Responses",
    exportCsv: "Export CSV",
    status: { sent: "Sent", completed: "Completed" },
    reason: { slot_taken: "Slot taken", invalid_booking: "No suitable time" },
    delete: "Delete",
    deactivateInstead: "This survey has responses — deactivate it instead of deleting.",
    assignLabel: "Survey on decline",
    assignNone: "None",
    empty: "No surveys yet.",
    questionsCount: "{count} questions",
  },
```

- [ ] **Step 2: Add the same keys to `ru.ts` and `kk.ts`**

Translate every key above into Russian (`ru.ts`) and Kazakh (`kk.ts`) with the same structure. Example `ru.ts` `nav.surveys: "Опросы"` and `surveys.create: "Создать опрос"`, `surveys.type: { single: "Один вариант", multi: "Несколько вариантов", rating: "Оценка", text: "Свободный текст" }`, etc. Keep keys identical across all three files (typecheck enforces parity).

- [ ] **Step 3: Verify parity + format**

Run: `pnpm --filter admin typecheck && pnpm format`
Expected: typecheck green (all three locales have identical key sets), formatter clean.

- [ ] **Step 4: Commit**

```bash
git add apps/admin/app/shared/i18n/dictionaries/
git commit -m "feat(surveys-admin): i18n strings (en/ru/kk)"
```

---

## Task 7: Full admin verification

- [ ] **Step 1: Run the full admin suite**

Run:
```bash
cd /Users/temirlan/Workspace/in-house/lead-cat
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin test && pnpm format:check
```
Expected: all green.

- [ ] **Step 2: Manual smoke (optional but recommended)**

Start the admin app, create a survey with one of each question type, assign it to a booking event-type, and confirm the responses page renders an empty table + CSV downloads.

- [ ] **Step 3: Commit any formatting fixes**

```bash
git add -A apps/admin
git commit -m "chore(surveys-admin): formatting + verification" || echo "nothing to commit"
```

---

## Self-review notes (addressed)

- **Spec coverage:** library list + builder (Tasks 1–4), 4 question types with conditional fields (Task 3), backend-mirrored validation (Task 2), responses table + filters + CSV (Task 5), assignment select on event-type (Task 5), nav + routes (Task 4), i18n en/ru/kk (Task 6).
- **Type consistency:** `SurveyInput`/`Survey`/`SurveyResponse` come from `@leadcat/api-client` via `entities/survey/types`; `surveyKeys` shape fixed in Task 1 and reused in hooks; `surveySchema`/`toSurveyInput` defined in Task 2 and consumed by the dialog in Task 3.
- **Ordering dependency:** Task 6 (i18n keys) must land before Tasks 3–5 typecheck cleanly; noted in Task 3 and Task 6.
- **Assumption to confirm:** the OpenAPI generator emits `Survey`, `SurveyQuestion`, `SurveyResponse`, `SurveyInput` named types. If not, define the input/response types locally in `entities/survey/types.ts` (fallback noted in Task 1).
