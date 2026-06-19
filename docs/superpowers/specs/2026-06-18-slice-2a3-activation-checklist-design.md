# Slice 2a-3 — First-Run Activation Checklist (design)

**Date:** 2026-06-18
**Status:** approved — ready for implementation plan
**Part of:** SaaS Product Completion epic, **Track 2a (activation/onboarding)**, sub-slice **3** (final). Follows 2a-1, 2a-2.

## Epic context

2a-1 fixed the onboarding gate + invite accept/decline; 2a-2 added join-by-slug requests. 2a-3
closes Track 2a (and the activation half of the epic) with a first-run **activation checklist**
on the admin dashboard that guides a new user to connect a calendar (Track 1), invite a
teammate, and create their first meeting.

## Goal

A new user landing on the dashboard sees a checklist of 3 setup steps with live done/todo
state and a CTA per step; it auto-hides once all are complete (or on manual dismiss).

## Decisions (from brainstorming)

- **3 steps:** connect a calendar, invite a teammate, create your first meeting.
- **Completion derived client-side** from existing queries — **no new backend**.
- **Dismissal via localStorage**; auto-hide when all 3 complete.
- **Admin web only** (mini-app checklist deferred).

## Background — verified current state

- Dashboard: `apps/admin/app/features/dashboard/pages/dashboard-page.tsx` — uses `useMe`,
  `useActiveOrg(me?.organizations)` → `{activeOrg, activeOrgId}`, `useMembers(activeOrgId)`,
  `useInvites(activeOrgId)`, renders `PageHeader` + a 3-up grid of `StatCard`s. Route
  `_app._index.tsx` → `<DashboardPage/>`.
- Existing hooks to derive completion (all already used elsewhere in admin):
  - `useCalendarConnections()` (1a, `entities/calendar-connection/queries.ts`) → `[{provider,
    connected, email, scopes}]`.
  - `useMembers(orgId)` / `useInvites(orgId)` (`entities/org/queries.ts`).
  - `useMeetings(orgId, filter?)` (`entities/meeting/queries.ts`).
- i18n: `shared/i18n/dictionaries/{en,ru,kk}.ts`, `useT()`, parity compile-enforced, admin
  formal. `@leadcat/ui` provides `Card`/`Button` + lucide icons (`Check`, `Circle`/`CircleDashed`,
  `Calendar`, `Users`, `CalendarPlus`, etc.).
- Navigation targets: Settings (calendar card) — the route the 1b/1a calendar card lives on
  (`/settings`); Invites page (`/invites`); meetings create (`/meetings` + create dialog, or the
  meetings route). Confirm exact paths from `routes.ts` during implementation.

## Design

### A. Step computation (pure)

`features/onboarding-checklist/lib/steps.ts` exports a pure function:
```ts
type ChecklistStep = { key: "calendar" | "invite" | "meeting"; done: boolean }
function computeSteps(input: {
  connections: { connected: boolean }[]
  membersCount: number
  invitesCount: number
  meetingsCount: number
}): ChecklistStep[]
```
- `calendar` done = `connections.some(c => c.connected)`.
- `invite` done = `invitesCount > 0 || membersCount > 1`.
- `meeting` done = `meetingsCount > 0`.
Pure + unit-testable (no React).

### B. Checklist card

`features/onboarding-checklist/components/activation-checklist.tsx`:
- Reads `useCalendarConnections()`, `useMembers(activeOrgId)`, `useInvites(activeOrgId)`,
  `useMeetings(activeOrgId ?? "")`; passes counts to `computeSteps`.
- Renders a `Card` titled "Get started" with each step as a row: a done/todo icon + label +
  (when not done) a CTA `Button`/link to the action route. A header progress line ("{n} of 3
  done") via a param key.
- **Hidden when:** all 3 steps done, OR the user dismissed it (localStorage key e.g.
  `lc_checklist_dismissed`), OR there is no `activeOrgId`. A "Dismiss" text button sets the flag.
- While the underlying queries are pending, render nothing (or a skeleton) — avoid flicker that
  shows steps as "todo" before data loads.
- ≤300 lines, no comments, no emoji (lucide only).

### C. Dashboard integration

In `dashboard-page.tsx`, render `<ActivationChecklist activeOrgId={activeOrgId} />` between the
`PageHeader` and the stat-card grid. No other dashboard changes.

### D. Dismissal persistence

A tiny `features/onboarding-checklist/lib/dismissed.ts`: `isChecklistDismissed()` /
`dismissChecklist()` over `localStorage` (guarded for SSR/`typeof window`). KISS — no backend.

### E. i18n

`dashboard.checklist.{title, progress, calendar, calendarCta, invite, inviteCta, meeting,
meetingCta, dismiss, allDone}` in en/ru/kk (formal RU). `progress` takes `{done}`/`{total}`
params.

## Testing / verification

- **Pure `computeSteps`** unit test (Vitest if configured in admin; else keep the function
  trivially correct + covered by typecheck). Cases: none done; calendar-only; all done;
  invite-via-members vs invite-via-invites.
  - NOTE: confirm whether admin has a test runner. If not, the pure function is still
    structured for testability; do not add a test runner in this slice (out of scope) — verify
    via typecheck/build + manual reasoning, and state that in the plan.
- **Frontend:** admin `typecheck`/`lint`/`build` green; i18n parity en/ru/kk.
- No backend changes → no Go build needed (but a final `git status` clean check).

## Risks & mitigations

- **Query loading flicker** (steps briefly show "todo"). *Mitigation:* render nothing until the
  relevant queries resolve.
- **`useMeetings` filter arg** — it may require a filter object. *Mitigation:* pass an empty/default
  filter; confirm the signature during implementation.
- **No test runner in admin.** *Mitigation:* keep `computeSteps` pure; if no Vitest, skip the
  unit test (don't introduce a runner here) and rely on typecheck/build.
- **localStorage SSR.** Admin is SPA (ssr:false), but guard `typeof window` anyway.

## Done criteria

- `computeSteps` pure helper + `ActivationChecklist` card + localStorage dismissal helper.
- Card on the dashboard above the stats; live done/todo from existing queries; auto-hide when
  complete or dismissed; CTA per incomplete step deep-links to the right route.
- i18n en/ru/kk; admin typecheck/lint/build green.
- No backend changes. Mini-app checklist explicitly deferred.
