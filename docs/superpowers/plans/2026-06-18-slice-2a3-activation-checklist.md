# Slice 2a-3 — First-Run Activation Checklist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A dashboard activation checklist (connect calendar / invite teammate / create first meeting) with live done/todo state, deep-link CTAs, and localStorage dismissal — frontend-only.

**Architecture:** A pure `computeSteps` helper + an `ActivationChecklist` card that derives completion from existing queries (`useCalendarConnections`, `useMembers`/`useInvites`, `useMeetings`) and self-hides when complete or dismissed. Mounted on the dashboard. No backend changes.

**Tech Stack:** React Router v7 / shadcn / TanStack Query admin SPA; `@leadcat/ui` (Card/Button + lucide).

## Global Constraints

- Admin app at `apps/admin`. Spec: `docs/superpowers/specs/2026-06-18-slice-2a3-activation-checklist-design.md`.
- Frontend-only — NO backend changes. Files ≤300 lines, no emoji (lucide only), no comments; i18n keys in en/ru/kk (admin **formal**), parity compile-enforced; never repo-wide prettier (additive edits); pnpm filter `admin`.
- Admin has **no test runner** — do NOT add one; verify `computeSteps` via typecheck + reasoning. Keep it pure for future testability.
- Work on `main`; never `git add -A`; **verify HEAD before each commit** (user commits in parallel — commit on top, never rebase/reset). Trailer: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

**Reference (verified):**
- Dashboard `features/dashboard/pages/dashboard-page.tsx`: `useMe`, `useActiveOrg(me?.organizations ?? [])` → `{activeOrg, activeOrgId}`, `useMembers(activeOrgId)`, `useInvites(activeOrgId)`, `PageHeader` + 3-up `StatCard` grid.
- Hooks: `useCalendarConnections()` (`entities/calendar-connection/queries.ts`) → `[{provider, connected, email, scopes}]`; `useMembers(orgId)`/`useInvites(orgId)` (`entities/org/queries.ts`); `useMeetings(orgId: string|null, filter={})` (`entities/meeting/queries.ts`, `enabled: Boolean(orgId)`).
- Routes (`routes.ts`): `/settings` (calendar card lives here), `/invites`, `/meetings`. Use `Link`/`useNavigate` from `react-router`.
- i18n dicts `shared/i18n/dictionaries/{en,ru,kk}.ts`; `useT()`.

---

### Task 1: Activation checklist (helpers + card + dashboard mount + i18n)

**Files:**
- Create: `apps/admin/app/features/onboarding-checklist/lib/steps.ts`
- Create: `apps/admin/app/features/onboarding-checklist/lib/dismissed.ts`
- Create: `apps/admin/app/features/onboarding-checklist/components/activation-checklist.tsx`
- Modify: `apps/admin/app/features/dashboard/pages/dashboard-page.tsx` — mount the card
- Modify: `apps/admin/app/shared/i18n/dictionaries/{en,ru,kk}.ts` — `dashboard.checklist.*`

**Interfaces:**
- Produces: `computeSteps(input) => ChecklistStep[]`; `isChecklistDismissed()`/`dismissChecklist()`; `<ActivationChecklist activeOrgId={string | null} />`.

- [ ] **Step 1: Pure step helper** — `lib/steps.ts`:
```ts
export type ChecklistStepKey = "calendar" | "invite" | "meeting"

export type ChecklistStep = { key: ChecklistStepKey; done: boolean }

export function computeSteps(input: {
  connections: { connected: boolean }[]
  membersCount: number
  invitesCount: number
  meetingsCount: number
}): ChecklistStep[] {
  return [
    { key: "calendar", done: input.connections.some((c) => c.connected) },
    { key: "invite", done: input.invitesCount > 0 || input.membersCount > 1 },
    { key: "meeting", done: input.meetingsCount > 0 },
  ]
}

export function allDone(steps: ChecklistStep[]): boolean {
  return steps.every((s) => s.done)
}

export function doneCount(steps: ChecklistStep[]): number {
  return steps.filter((s) => s.done).length
}
```

- [ ] **Step 2: Dismissal helper** — `lib/dismissed.ts`:
```ts
const KEY = "lc_checklist_dismissed"

export function isChecklistDismissed(): boolean {
  if (typeof window === "undefined") {
    return false
  }
  return window.localStorage.getItem(KEY) === "1"
}

export function dismissChecklist(): void {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(KEY, "1")
}
```

- [ ] **Step 3: Checklist card** — `components/activation-checklist.tsx`. Compute steps from the hooks; hide when no org, all done, or dismissed; per-step CTA to the route. Skeleton/null while loading. Use a local `useState` seeded from `isChecklistDismissed()` so the manual dismiss hides it immediately.
```tsx
import {
  Button,
  Calendar,
  CalendarPlus,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Check,
  Circle,
  Users,
  type LucideIcon,
} from "@leadcat/ui"
import { useState } from "react"
import { Link } from "react-router"

import { useCalendarConnections } from "~/entities/calendar-connection/queries"
import { useMeetings } from "~/entities/meeting/queries"
import { useInvites, useMembers } from "~/entities/org/queries"
import { useT } from "~/shared/i18n/context"
import {
  allDone,
  computeSteps,
  doneCount,
  type ChecklistStepKey,
} from "../lib/steps"
import { dismissChecklist, isChecklistDismissed } from "../lib/dismissed"

const META: Record<ChecklistStepKey, { icon: LucideIcon; to: string }> = {
  calendar: { icon: Calendar, to: "/settings" },
  invite: { icon: Users, to: "/invites" },
  meeting: { icon: CalendarPlus, to: "/meetings" },
}

export function ActivationChecklist({
  activeOrgId,
}: {
  activeOrgId: string | null
}) {
  const t = useT()
  const [dismissed, setDismissed] = useState(isChecklistDismissed)
  const connections = useCalendarConnections()
  const members = useMembers(activeOrgId)
  const invites = useInvites(activeOrgId)
  const meetings = useMeetings(activeOrgId)

  if (!activeOrgId || dismissed) {
    return null
  }
  if (
    connections.isPending ||
    members.isPending ||
    invites.isPending ||
    meetings.isPending
  ) {
    return null
  }

  const steps = computeSteps({
    connections: connections.data ?? [],
    membersCount: members.data?.length ?? 0,
    invitesCount: invites.data?.length ?? 0,
    meetingsCount: meetings.data?.length ?? 0,
  })
  if (allDone(steps)) {
    return null
  }

  function onDismiss() {
    dismissChecklist()
    setDismissed(true)
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-3">
        <div>
          <CardTitle>{t("dashboard.checklist.title")}</CardTitle>
          <p className="mt-1 text-sm text-muted-foreground">
            {t("dashboard.checklist.progress", {
              done: doneCount(steps),
              total: steps.length,
            })}
          </p>
        </div>
        <Button variant="ghost" size="sm" onClick={onDismiss}>
          {t("dashboard.checklist.dismiss")}
        </Button>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {steps.map((step) => {
          const Icon = step.done ? Check : META[step.key].icon
          return (
            <div
              key={step.key}
              className="flex items-center justify-between gap-3"
            >
              <span className="flex items-center gap-2">
                <Icon
                  className={
                    step.done ? "size-4 text-primary" : "size-4 text-muted-foreground"
                  }
                />
                <span className={step.done ? "text-muted-foreground line-through" : ""}>
                  {t(`dashboard.checklist.${step.key}`)}
                </span>
              </span>
              {step.done ? null : (
                <Button asChild size="sm" variant="outline">
                  <Link to={META[step.key].to}>
                    {t(`dashboard.checklist.${step.key}Cta`)}
                  </Link>
                </Button>
              )}
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
```
(If `@leadcat/ui` does not re-export a `Circle`/`Check`/`CalendarPlus` icon, import the lucide icon names it DOES export — check the package's exports; the icon choice is cosmetic. `Button asChild` + `Link` is the shadcn pattern; if `asChild` isn't supported, use `onClick={() => navigate(to)}` with `useNavigate`.)

- [ ] **Step 4: Mount on dashboard** — in `dashboard-page.tsx`, import `ActivationChecklist` and render it between `<PageHeader/>` and the `<div className="grid ...">` stat grid:
```tsx
<ActivationChecklist activeOrgId={activeOrgId} />
```

- [ ] **Step 5: i18n** — add to en/ru/kk under `dashboard.checklist`:
```
title        EN "Get started"
progress     EN "{done} of {total} done"
calendar     EN "Connect your calendar"
calendarCta  EN "Connect"
invite       EN "Invite a teammate"
inviteCta    EN "Invite"
meeting      EN "Create your first meeting"
meetingCta   EN "Create"
dismiss      EN "Dismiss"
```
Provide formal RU + KK for every key, in all three dicts (parity compile-enforced). `progress` uses `{done}`/`{total}` interpolation (match the dict's existing param convention).

- [ ] **Step 6: Verify + commit**
```bash
pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build
git add apps/admin/app/features/onboarding-checklist apps/admin/app/features/dashboard/pages/dashboard-page.tsx apps/admin/app/shared/i18n/dictionaries
git commit -m "feat(admin): first-run activation checklist on dashboard + i18n"
```

---

### Task 2: Whole-slice verification

**Files:** none

- [ ] **Step 1: Frontend** — `pnpm --filter admin typecheck && pnpm --filter admin lint && pnpm --filter admin build`. Green; i18n parity en/ru/kk holds (compile-enforced).
- [ ] **Step 2: Behavior reasoning (documented)** — confirm: checklist hides when `activeOrgId` is null, when all 3 steps are done, or when dismissed (localStorage); each incomplete step links to `/settings`/`/invites`/`/meetings`; nothing renders while queries are pending (no flicker).
- [ ] **Step 3: No backend touched** — `git status` shows only the intended admin files; no `apps/backend` changes; user parallel WIP untouched.

---

## Notes for the executor

- **No backend, no test runner** — verify via typecheck/lint/build; keep `computeSteps` pure.
- **Loading flicker:** render `null` until all four queries resolve, so steps never briefly show as "todo".
- **Icon/`asChild` fallbacks:** use whatever `@leadcat/ui` actually exports; the icon + link mechanism is cosmetic, not load-bearing.
- **Deferred:** mini-app checklist; any backend onboarding-state persistence.
