# Survey-on-decline Phase 1 — Landing Card Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Advertise the survey-on-decline feature on the landing page with one localized feature card (en/ru/kk).

**Architecture:** The landing (`apps/landing`) is a localized SPA. `en.ts` is the source dictionary (`satisfies Dict`, exported as `LandingDict = typeof en`); `ru.ts` and `kk.ts` are typed `: LandingDict`, so the type system forces all three to have identical shape. The `features` section renders `features.items[]` (an array of `{ title, body }`) — adding a card means appending one item to all three locales.

**Tech Stack:** React 19, TypeScript, the landing's `features` section component.

## Global Constraints

- App root: `apps/landing`. Dictionaries: `app/shared/i18n/dictionaries/{en,ru,kk}.ts`.
- All three locales must have identical structure (the `LandingDict` type enforces this — a missing/extra item field fails `typecheck`).
- Marketing copy **is** localized (unlike operator survey content).
- `pnpm --filter landing typecheck && pnpm --filter landing lint && pnpm --filter landing build` clean before commit; `pnpm format` clean.

---

## File map

Modify:
- `apps/landing/app/shared/i18n/dictionaries/en.ts` — append a `features.items` card.
- `apps/landing/app/shared/i18n/dictionaries/ru.ts` — same card, Russian.
- `apps/landing/app/shared/i18n/dictionaries/kk.ts` — same card, Kazakh.
- (If needed) the features section component — only if the grid hardcodes a column count that breaks with 5 cards.

---

## Task 1: Add the feature card (all three locales)

**Files:**
- Modify: `apps/landing/app/shared/i18n/dictionaries/en.ts`
- Modify: `apps/landing/app/shared/i18n/dictionaries/ru.ts`
- Modify: `apps/landing/app/shared/i18n/dictionaries/kk.ts`

**Interfaces:** appends one `{ title, body }` object to the existing `features.items` array in each locale.

- [ ] **Step 1: Append the card to `en.ts`**

In `en.ts`, inside `features.items: [ ... ]`, add as the last item:

```ts
      {
        title: "Never lose a declined lead",
        body: "When a time doesn't fit, Lead Cat offers your visitor a short survey you built — capture the why, the feedback, and the lead, automatically.",
      },
```

- [ ] **Step 2: Append the matching card to `ru.ts`**

In `ru.ts`, same position in `features.items`:

```ts
      {
        title: "Не теряйте лида при отказе",
        body: "Когда время не подошло, Lead Cat предложит посетителю короткий опрос, который вы собрали сами — соберите причину, обратную связь и контакт автоматически.",
      },
```

- [ ] **Step 3: Append the matching card to `kk.ts`**

In `kk.ts`, same position in `features.items`:

```ts
      {
        title: "Бас тартқан лидті жоғалтпаңыз",
        body: "Уақыт сәйкес келмегенде, Lead Cat келушіге өзіңіз жинаған қысқа сауалнаманы ұсынады — себебін, пікірін және байланысын автоматты түрде жинаңыз.",
      },
```

- [ ] **Step 4: Verify type parity**

Run: `pnpm --filter landing typecheck`
Expected: PASS — all three locales have the same `features.items` element shape (`LandingDict` enforces parity; a mismatch errors at `ru.ts`/`kk.ts`).

- [ ] **Step 5: Commit**

```bash
git add apps/landing/app/shared/i18n/dictionaries/
git commit -m "feat(landing): advertise survey-on-decline feature (en/ru/kk)"
```

---

## Task 2: Confirm the grid renders 5 cards cleanly

**Files:**
- Possibly modify: the landing `features` section component (only if needed).

**Interfaces:** none — visual confirmation that the grid handles 5 items.

- [ ] **Step 1: Locate the features section renderer**

Run: `grep -rn "features.items\|\.items\b" apps/landing/app --include=*.tsx`
Read the component that maps `features.items`. Check the grid classes (e.g. `grid-cols-2`/`md:grid-cols-4`).

- [ ] **Step 2: Adjust the grid only if 5 cards break the layout**

If the grid hardcodes exactly 4 columns and 5 cards leave a lone orphan that looks broken, adjust the responsive classes (e.g. allow `lg:grid-cols-3` so 5 wraps cleanly, or `md:grid-cols-2`). Make the **minimal** class change; do not restructure the section. If the grid already flows (auto-fit / wraps), make no change.

- [ ] **Step 3: Build + lint**

Run: `pnpm --filter landing typecheck && pnpm --filter landing lint && pnpm --filter landing build`
Expected: all green.

- [ ] **Step 4: Manual visual check (recommended)**

Start the landing app, scroll to the features section, confirm the new card renders in all three languages (toggle the language switcher) and the grid looks intentional.

- [ ] **Step 5: Commit any grid tweak**

```bash
git add apps/landing/app
git commit -m "fix(landing): tidy features grid for the fifth card" || echo "no grid change needed"
```

---

## Self-review notes (addressed)

- **Spec coverage:** one localized feature card in the existing `features` section (Task 1), all three locales (Task 1), grid integrity (Task 2). Marketing copy localized; no new section (KISS, per the approved design).
- **Type consistency:** the card is a `{ title, body }` object matching every other `features.items` element; `LandingDict` parity enforced by typecheck (Task 1 Step 4).
- **Risk:** the only layout risk is a 4-column grid with a lone 5th card — Task 2 handles it with a minimal, conditional class change.
