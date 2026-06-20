# WS3 — Frontend Unit Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up Vitest in the frontend monorepo as a cached turbo `test` task + PR gate, and cover the pure-logic helpers across `packages/ui`, `apps/mini-app`, `apps/admin`, and `apps/landing` with fast, DOM-free unit tests.

**Architecture:** Per-package standalone `vitest.config.ts` using `vite-tsconfig-paths` to resolve the `~/*` alias from each package's `tsconfig.json`; `environment: 'node'` (pure functions only, no jsdom). A turbo `test` task wires the per-package `vitest run` scripts; `_build.yml` runs it as a red-on-fail gate.

**Tech Stack:** Vitest, vite-tsconfig-paths, Vite 8 (already present), pnpm workspaces, turbo.

## Global Constraints

- **Node env only:** every `vitest.config.ts` sets `test.environment: "node"`. No jsdom, no React Testing Library, no DOM/component tests in this slice.
- **Pure logic only:** tests assert deterministic input→output. No network, no real timers, no `localStorage`. Functions that read `new Date()` / system locale / timezone are tested only via their deterministic branches or with explicit `locale`/`timeZone`/`siteUrl` arguments.
- **Colocation:** each test file is `<source>.test.ts` next to the source file (mirrors the backend `_test.go` convention).
- **Install via pnpm, not hand-edited versions:** add devDeps with `pnpm --filter <dir> add -D <pkg>` so pnpm resolves versions compatible with Vite 8 / the existing lockfile. Use directory filters (`./apps/admin`, `./packages/ui`) to avoid package-name ambiguity.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Bootstrap Vitest in `packages/ui` + turbo `test` task

Proves the runner end-to-end in one package and registers the turbo task the later tasks rely on.

**Files:**
- Modify: `turbo.json` (add `test` task)
- Modify: `packages/ui/package.json` (add `test` script + devDeps)
- Create: `packages/ui/vitest.config.ts`
- Test: `packages/ui/src/lib/date.test.ts`

**Interfaces:**
- Produces: a turbo `test` task and a working per-package Vitest pattern that Tasks 2–4 copy.

- [ ] **Step 1: Add devDependencies to `packages/ui`**

Run:
```bash
pnpm --filter ./packages/ui add -D vitest vite-tsconfig-paths
```
Expected: lockfile updates; `vitest` + `vite-tsconfig-paths` appear under `packages/ui/package.json` `devDependencies`.

- [ ] **Step 2: Add the `test` script to `packages/ui/package.json`**

In `packages/ui/package.json` `"scripts"`, add:
```json
    "test": "vitest run"
```

- [ ] **Step 3: Create `packages/ui/vitest.config.ts`**

```ts
import { defineConfig } from "vitest/config"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
})
```

- [ ] **Step 4: Write the failing test `packages/ui/src/lib/date.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import {
  parseIsoDate,
  formatIsoDate,
  parseTimeValue,
  formatTimeValue,
  addMinutesToTime,
  timeToMinutes,
  diffMinutes,
  buildTimeOptions,
  dateFnsLocaleFromCode,
  dayPickerLocaleFromCode,
  DEFAULT_MEETING_DURATION_MIN,
} from "./date"

describe("parse/format iso date", () => {
  it("round-trips a valid iso date", () => {
    expect(formatIsoDate(parseIsoDate("2026-06-20")!)).toBe("2026-06-20")
  })
  it("returns undefined for empty or garbage input", () => {
    expect(parseIsoDate(undefined)).toBeUndefined()
    expect(parseIsoDate("not-a-date")).toBeUndefined()
  })
})

describe("time value parsing", () => {
  it("parses HH:MM", () => {
    expect(parseTimeValue("09:30")).toEqual({ hour: 9, minute: 30 })
  })
  it("falls back to 0:0 for empty or garbage", () => {
    expect(parseTimeValue(undefined)).toEqual({ hour: 0, minute: 0 })
    expect(parseTimeValue("xx:yy")).toEqual({ hour: 0, minute: 0 })
  })
  it("formats with zero-padding", () => {
    expect(formatTimeValue(9, 5)).toBe("09:05")
  })
  it("timeToMinutes", () => {
    expect(timeToMinutes("01:30")).toBe(90)
  })
})

describe("addMinutesToTime", () => {
  it("adds within the day", () => {
    expect(addMinutesToTime("09:00", 90)).toBe("10:30")
  })
  it("wraps past midnight", () => {
    expect(addMinutesToTime("23:30", 60)).toBe("00:30")
  })
  it("wraps negative", () => {
    expect(addMinutesToTime("00:15", -30)).toBe("23:45")
  })
})

describe("diffMinutes", () => {
  it("returns positive delta", () => {
    expect(diffMinutes("09:00", "10:00")).toBe(60)
  })
  it("falls back to default for non-positive or missing", () => {
    expect(diffMinutes("10:00", "09:00")).toBe(DEFAULT_MEETING_DURATION_MIN)
    expect(diffMinutes("", "10:00")).toBe(DEFAULT_MEETING_DURATION_MIN)
  })
})

describe("buildTimeOptions", () => {
  it("produces a full grid for the default step", () => {
    expect(buildTimeOptions().length).toBe((24 * 60) / 5)
    expect(buildTimeOptions()[0]).toBe("00:00")
  })
  it("honors a custom step", () => {
    expect(buildTimeOptions(60).length).toBe(24)
  })
})

describe("locale resolution", () => {
  it("maps ru and kk to the ru locale", () => {
    expect(dateFnsLocaleFromCode("ru").code).toBe("ru")
    expect(dateFnsLocaleFromCode("kk").code).toBe("ru")
    expect(dayPickerLocaleFromCode("kk").code).toBe("ru")
  })
  it("defaults to en", () => {
    expect(dateFnsLocaleFromCode(undefined).code).toBe("en-US")
    expect(dateFnsLocaleFromCode("en").code).toBe("en-US")
  })
})
```

> Note: `date-fns` locale objects expose a `.code` (`"ru"`, `"en-US"`). If `react-day-picker/locale`'s object exposes a different field, assert identity instead (`expect(dayPickerLocaleFromCode("kk")).toBe(ru)` after importing `ru` from `react-day-picker/locale`). Adjust during the red run if `.code` is undefined.

- [ ] **Step 5: Run the test and watch it run (it should pass once config resolves)**

Run:
```bash
pnpm --filter ./packages/ui test
```
Expected: Vitest discovers `src/lib/date.test.ts` and all assertions PASS. If the `.code` assertion fails, fix per the note in Step 4 and re-run.

- [ ] **Step 6: Add the `test` task to `turbo.json`**

In `turbo.json` `"tasks"`, add:
```json
    "test": {
      "outputs": []
    },
```

- [ ] **Step 7: Verify the task runs through turbo**

Run:
```bash
pnpm turbo run test --filter=./packages/ui
```
Expected: turbo runs `@leadcat/ui#test`, suite PASSES.

- [ ] **Step 8: Commit**

```bash
git add turbo.json packages/ui/package.json packages/ui/vitest.config.ts packages/ui/src/lib/date.test.ts pnpm-lock.yaml
git commit -m "$(cat <<'EOF'
test(ws3): bootstrap Vitest in packages/ui + turbo test task

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: `apps/mini-app` suites

**Files:**
- Modify: `apps/mini-app/package.json` (add `test` script + devDeps)
- Create: `apps/mini-app/vitest.config.ts`
- Test: `apps/mini-app/app/shared/lib/format.test.ts`
- Test: `apps/mini-app/app/shared/lib/display-name.test.ts`
- Test: `apps/mini-app/app/features/meetings/lib/group-series.test.ts`

**Interfaces:**
- Consumes: the turbo `test` task from Task 1.

- [ ] **Step 1: Add devDependencies + script**

Run:
```bash
pnpm --filter ./apps/mini-app add -D vitest vite-tsconfig-paths
```
Then add to `apps/mini-app/package.json` `"scripts"`:
```json
    "test": "vitest run"
```

- [ ] **Step 2: Create `apps/mini-app/vitest.config.ts`**

```ts
import { defineConfig } from "vitest/config"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    environment: "node",
    include: ["app/**/*.test.ts"],
  },
})
```

- [ ] **Step 3: Write `apps/mini-app/app/shared/lib/format.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { formatDate, formatDateLong, formatTimeRange, addDaysIso } from "./format"

describe("formatDate", () => {
  it("formats a valid iso date in the given locale", () => {
    const out = formatDate("2026-06-20", "en")
    expect(out).toContain("Jun")
    expect(out).toContain("20")
  })
  it("returns the input unchanged for an invalid iso", () => {
    expect(formatDate("nope")).toBe("nope")
  })
})

describe("formatDateLong", () => {
  it("includes the year", () => {
    expect(formatDateLong("2026-06-20", "en")).toContain("2026")
  })
  it("returns the input unchanged for an invalid iso", () => {
    expect(formatDateLong("")).toBe("")
  })
})

describe("formatTimeRange", () => {
  it("joins start and end", () => {
    expect(formatTimeRange("09:00", "10:00")).toBe("09:00 – 10:00")
  })
  it("returns start alone when end is empty", () => {
    expect(formatTimeRange("09:00", "")).toBe("09:00")
  })
  it("returns empty string when start is empty", () => {
    expect(formatTimeRange("", "10:00")).toBe("")
  })
})

describe("addDaysIso", () => {
  it("adds days within a month", () => {
    expect(addDaysIso("2026-06-20", 5)).toBe("2026-06-25")
  })
  it("rolls over month boundaries", () => {
    expect(addDaysIso("2026-06-30", 1)).toBe("2026-07-01")
  })
  it("subtracts days", () => {
    expect(addDaysIso("2026-07-01", -1)).toBe("2026-06-30")
  })
})
```

- [ ] **Step 4: Write `apps/mini-app/app/shared/lib/display-name.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { getGreetingName } from "./display-name"

describe("getGreetingName", () => {
  it("prefers a non-empty telegram first name", () => {
    expect(getGreetingName("Иванов Иван Иванович", "Vanya")).toBe("Vanya")
  })
  it("falls back to the given name (second FIO token)", () => {
    expect(getGreetingName("Иванов Иван Иванович")).toBe("Иван")
  })
  it("uses the single token when only one is present", () => {
    expect(getGreetingName("Иван")).toBe("Иван")
  })
  it("returns empty string for empty/undefined input", () => {
    expect(getGreetingName(undefined)).toBe("")
    expect(getGreetingName("   ", "  ")).toBe("")
  })
})
```

- [ ] **Step 5: Write `apps/mini-app/app/features/meetings/lib/group-series.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { groupBySeries } from "./group-series"
import type { Meeting } from "~/entities/meeting/types"

// Minimal factory — only the fields groupBySeries reads matter.
const mk = (id: string, series_id?: string) =>
  ({ id, series_id }) as unknown as Meeting

describe("groupBySeries", () => {
  it("returns [] for empty input", () => {
    expect(groupBySeries([])).toEqual([])
  })
  it("treats meetings without a series_id as singles", () => {
    const result = groupBySeries([mk("a"), mk("b")])
    expect(result).toHaveLength(2)
    expect(result.every((g) => g.kind === "single")).toBe(true)
  })
  it("groups by series in first-seen order, with singles appended after", () => {
    const result = groupBySeries([
      mk("s1a", "s1"),
      mk("solo"),
      mk("s2a", "s2"),
      mk("s1b", "s1"),
    ])
    expect(result.map((g) => g.kind)).toEqual(["series", "series", "single"])
    const first = result[0]
    expect(first.kind === "series" && first.seriesId).toBe("s1")
    expect(first.kind === "series" && first.meetings.map((m) => m.id)).toEqual([
      "s1a",
      "s1b",
    ])
  })
})
```

- [ ] **Step 6: Run the mini-app suite**

Run:
```bash
pnpm --filter ./apps/mini-app test
```
Expected: three test files discovered, all PASS.

> If `@leadcat/ui` (re-exported by `format.ts`) fails to resolve, confirm the package's `exports`/source entry is a `.ts` Vite can load; Vite handles TS sources directly, so no build step is needed.

- [ ] **Step 7: Commit**

```bash
git add apps/mini-app/package.json apps/mini-app/vitest.config.ts apps/mini-app/app pnpm-lock.yaml
git commit -m "$(cat <<'EOF'
test(ws3): mini-app pure-logic suites (format, display-name, group-series)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 3: `apps/admin` suites

**Files:**
- Modify: `apps/admin/package.json` (add `test` script + devDeps)
- Create: `apps/admin/vitest.config.ts`
- Test: `apps/admin/app/features/dashboard/lib/checklist-steps.test.ts`
- Test: `apps/admin/app/features/meetings/lib/format.test.ts`
- Test: `apps/admin/app/features/meetings/lib/group-series.test.ts`

- [ ] **Step 1: Add devDependencies + script**

Run:
```bash
pnpm --filter ./apps/admin add -D vitest vite-tsconfig-paths
```
Then add to `apps/admin/package.json` `"scripts"`:
```json
    "test": "vitest run"
```

- [ ] **Step 2: Create `apps/admin/vitest.config.ts`**

```ts
import { defineConfig } from "vitest/config"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    environment: "node",
    include: ["app/**/*.test.ts"],
  },
})
```

- [ ] **Step 3: Write `apps/admin/app/features/dashboard/lib/checklist-steps.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { computeSteps, allDone, doneCount } from "./checklist-steps"

const base = {
  connections: [] as { connected: boolean }[],
  membersCount: 1,
  invitesCount: 0,
  meetingsCount: 0,
}

describe("computeSteps", () => {
  it("marks calendar done when any connection is connected", () => {
    const steps = computeSteps({
      ...base,
      connections: [{ connected: false }, { connected: true }],
    })
    expect(steps.find((s) => s.key === "calendar")!.done).toBe(true)
  })
  it("marks invite done when there are invites OR more than one member", () => {
    expect(
      computeSteps({ ...base, invitesCount: 1 }).find((s) => s.key === "invite")!.done,
    ).toBe(true)
    expect(
      computeSteps({ ...base, membersCount: 2 }).find((s) => s.key === "invite")!.done,
    ).toBe(true)
    expect(
      computeSteps(base).find((s) => s.key === "invite")!.done,
    ).toBe(false)
  })
  it("marks meeting done when there is at least one meeting", () => {
    expect(
      computeSteps({ ...base, meetingsCount: 3 }).find((s) => s.key === "meeting")!.done,
    ).toBe(true)
  })
})

describe("allDone / doneCount", () => {
  it("counts and reports completion", () => {
    const steps = computeSteps({
      connections: [{ connected: true }],
      membersCount: 2,
      invitesCount: 0,
      meetingsCount: 1,
    })
    expect(doneCount(steps)).toBe(3)
    expect(allDone(steps)).toBe(true)
  })
  it("is not all done when a step is incomplete", () => {
    expect(allDone(computeSteps(base))).toBe(false)
  })
})
```

- [ ] **Step 4: Write `apps/admin/app/features/meetings/lib/format.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { formatMeetingDate, formatDateTime, meetingTitle } from "./format"
import type { Meeting } from "~/entities/meeting/types"

const mk = (partial: Partial<Meeting>) => partial as unknown as Meeting

describe("formatMeetingDate", () => {
  it("formats a valid date with explicit locale + timeZone", () => {
    const out = formatMeetingDate("2026-06-20T09:00:00Z", {
      locale: "en-US",
      timeZone: "UTC",
    })
    expect(out).toContain("Jun")
    expect(out).toContain("20")
  })
  it("returns the em dash for an invalid date", () => {
    expect(formatMeetingDate("not-a-date")).toBe("—")
  })
})

describe("formatDateTime", () => {
  it("returns the em dash for an invalid date", () => {
    expect(formatDateTime("not-a-date")).toBe("—")
  })
})

describe("meetingTitle", () => {
  it("uses the trimmed type when present", () => {
    expect(meetingTitle(mk({ type: "  Standup  ", name: "X | Y" }), "fb")).toBe(
      "Standup",
    )
  })
  it("falls back to the second pipe segment of name", () => {
    expect(meetingTitle(mk({ type: "  ", name: "Team | Sync" }), "fb")).toBe("Sync")
  })
  it("falls back to the provided fallback when nothing usable", () => {
    expect(meetingTitle(mk({ type: "", name: "OnlyOne" }), "fb")).toBe("fb")
  })
})
```

- [ ] **Step 5: Write `apps/admin/app/features/meetings/lib/group-series.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { groupBySeries } from "./group-series"
import type { Meeting } from "~/entities/meeting/types"

const mk = (id: string, series_id?: string) =>
  ({ id, series_id }) as unknown as Meeting

describe("groupBySeries (admin)", () => {
  it("returns [] for empty input", () => {
    expect(groupBySeries([])).toEqual([])
  })
  it("groups by series in first-seen order with singles appended", () => {
    const result = groupBySeries([mk("s1a", "s1"), mk("solo"), mk("s1b", "s1")])
    expect(result.map((g) => g.kind)).toEqual(["series", "single"])
    const first = result[0]
    expect(first.kind === "series" && first.meetings.map((m) => m.id)).toEqual([
      "s1a",
      "s1b",
    ])
  })
})
```

- [ ] **Step 6: Run the admin suite**

Run:
```bash
pnpm --filter ./apps/admin test
```
Expected: three test files discovered, all PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/admin/package.json apps/admin/vitest.config.ts apps/admin/app pnpm-lock.yaml
git commit -m "$(cat <<'EOF'
test(ws3): admin pure-logic suites (checklist-steps, meetings format/group-series)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 4: `apps/landing` suites

**Files:**
- Modify: `apps/landing/package.json` (add `test` script + devDeps)
- Create: `apps/landing/vitest.config.ts`
- Test: `apps/landing/app/shared/i18n/locale-path.test.ts`
- Test: `apps/landing/app/shared/i18n/locale-request.test.ts`
- Test: `apps/landing/app/shared/i18n/translate.test.ts`
- Test: `apps/landing/app/shared/seo/landing-meta.test.ts`

- [ ] **Step 1: Add devDependencies + script**

Run:
```bash
pnpm --filter ./apps/landing add -D vitest vite-tsconfig-paths
```
Then add to `apps/landing/package.json` `"scripts"`:
```json
    "test": "vitest run"
```

- [ ] **Step 2: Create `apps/landing/vitest.config.ts`**

```ts
import { defineConfig } from "vitest/config"
import tsconfigPaths from "vite-tsconfig-paths"

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    environment: "node",
    include: ["app/**/*.test.ts"],
  },
})
```

- [ ] **Step 3: Write `apps/landing/app/shared/i18n/locale-path.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { localePath } from "./locale-path"

describe("localePath", () => {
  it("returns root for the default locale (ru)", () => {
    expect(localePath("ru")).toBe("/")
    expect(localePath("ru", "#features")).toBe("/#features")
  })
  it("prefixes non-default locales", () => {
    expect(localePath("en")).toBe("/en")
    expect(localePath("kk", "#pricing")).toBe("/kk#pricing")
  })
})
```

- [ ] **Step 4: Write `apps/landing/app/shared/i18n/locale-request.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { parseUrlLocale, localeCookieHeader } from "./locale-request"

describe("parseUrlLocale", () => {
  it("accepts url locales", () => {
    expect(parseUrlLocale("en")).toBe("en")
    expect(parseUrlLocale("kk")).toBe("kk")
  })
  it("rejects the default locale (ru is not a url locale) and garbage", () => {
    expect(parseUrlLocale("ru")).toBeNull()
    expect(parseUrlLocale("xx")).toBeNull()
    expect(parseUrlLocale(undefined)).toBeNull()
  })
})

describe("localeCookieHeader", () => {
  it("builds a year-long lax cookie", () => {
    const header = localeCookieHeader("en")
    expect(header).toContain("leadcat_locale=en")
    expect(header).toContain("Path=/")
    expect(header).toContain("Max-Age=31536000")
    expect(header).toContain("SameSite=Lax")
  })
})
```

- [ ] **Step 5: Write `apps/landing/app/shared/i18n/translate.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import { translate } from "./translate"
import type { Dict } from "./types"

const active: Dict = { nav: { home: "Главная" }, hi: "Привет, {name}" }
const fallback: Dict = { nav: { home: "Home", about: "About" } }

describe("translate", () => {
  it("looks up a nested key in the active dict", () => {
    expect(translate(active, fallback, "nav.home")).toBe("Главная")
  })
  it("falls back when the key is missing in active", () => {
    expect(translate(active, fallback, "nav.about")).toBe("About")
  })
  it("returns the key itself when missing in both", () => {
    expect(translate(active, fallback, "nav.missing")).toBe("nav.missing")
  })
  it("interpolates params, leaving unknown placeholders intact", () => {
    expect(translate(active, fallback, "hi", { name: "Иван" })).toBe("Привет, Иван")
    expect(translate(active, fallback, "hi")).toBe("Привет, {name}")
  })
})
```

- [ ] **Step 6: Write `apps/landing/app/shared/seo/landing-meta.test.ts`**

```ts
import { describe, expect, it } from "vitest"
import {
  landingCanonicalPath,
  sitemapXml,
  robotsTxt,
  landingMeta,
} from "./landing-meta"

const SITE = "https://example.test"

describe("landingCanonicalPath", () => {
  it("maps locales to their canonical path", () => {
    expect(landingCanonicalPath("ru")).toBe("/")
    expect(landingCanonicalPath("en")).toBe("/en")
  })
})

describe("sitemapXml", () => {
  it("lists the localized roots with absolute URLs", () => {
    const xml = sitemapXml(SITE)
    expect(xml).toContain("<loc>https://example.test</loc>")
    expect(xml).toContain("<loc>https://example.test/en</loc>")
    expect(xml).toContain("<loc>https://example.test/kk</loc>")
    expect(xml.startsWith("<?xml")).toBe(true)
  })
})

describe("robotsTxt", () => {
  it("points at the absolute sitemap URL", () => {
    expect(robotsTxt(SITE)).toContain("Sitemap: https://example.test/sitemap.xml")
  })
})

describe("landingMeta", () => {
  it("emits a canonical link and html lang for the locale", () => {
    const tags = landingMeta("en", { siteUrl: SITE })
    expect(tags).toContainEqual({ html: { lang: "en" } })
    expect(tags).toContainEqual({
      tagName: "link",
      rel: "canonical",
      href: "https://example.test/en",
    })
  })
})
```

> `landingMeta`/`sitemapXml`/`robotsTxt` accept an explicit `siteUrl`, so no `@leadcat/brand` `resolveSiteUrl()` env is needed. `dictionaries[locale]` is a static import — title/description come from the real dictionary; the test only asserts the structural tags (`html.lang`, canonical href), not copy.

- [ ] **Step 7: Run the landing suite**

Run:
```bash
pnpm --filter ./apps/landing test
```
Expected: four test files discovered, all PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/landing/package.json apps/landing/vitest.config.ts apps/landing/app pnpm-lock.yaml
git commit -m "$(cat <<'EOF'
test(ws3): landing pure-logic suites (i18n + seo meta)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 5: Wire the CI gate

**Files:**
- Modify: `.github/workflows/_build.yml` (frontend job turbo line)

- [ ] **Step 1: Add `test` to the frontend turbo run and include `packages/ui`**

In `.github/workflows/_build.yml`, the `frontend` job's final step currently is:
```yaml
      - run: pnpm turbo run typecheck lint build --filter=./apps/*
```
Change it to:
```yaml
      - run: pnpm turbo run typecheck lint build test --filter=./apps/* --filter=./packages/ui
```

- [ ] **Step 2: Verify the full frontend pipeline locally**

Run:
```bash
pnpm turbo run typecheck lint build test --filter=./apps/* --filter=./packages/ui
```
Expected: all four packages' `test` tasks run and PASS; `typecheck`, `lint`, `build` stay green.

- [ ] **Step 3: Prove the gate fails red on a broken helper**

Temporarily break one assertion's source (e.g. flip a return in `packages/ui/src/lib/date.ts` `formatTimeValue`), then run:
```bash
pnpm turbo run test --filter=./packages/ui
```
Expected: FAIL. Revert the change immediately afterward and re-run to confirm GREEN.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/_build.yml
git commit -m "$(cat <<'EOF'
ci(ws3): gate PRs on frontend unit tests (turbo test)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- Runner (Vitest, node env, approach A) → Tasks 1–4 each create a standalone `vitest.config.ts` with `vite-tsconfig-paths`. ✓
- turbo `test` task → Task 1 Step 6. ✓
- Per-package `test` script + devDeps → Tasks 1–4 Step 1. ✓
- CI gate with `packages/ui` in filter → Task 5. ✓
- mini-app targets (format, display-name, group-series) → Task 2. ✓
- admin targets (checklist-steps, meetings format/group-series) → Task 3. ✓
- packages/ui date.ts → Task 1. ✓
- landing targets (locale-path, locale-request, translate, landing-meta) → Task 4. ✓
- Out-of-scope (checklist-dismissed, motion.ts, component tests) → not implemented, per spec. ✓

**Placeholder scan:** No TODO/TBD; every code step shows complete test/config code. The two `> Note` blocks describe concrete fallback assertions, not deferred work. ✓

**Type consistency:** `groupBySeries` returns `MeetingGroup[]` with `kind`/`seriesId`/`meetings` — asserted consistently in Tasks 2 & 3. `computeSteps` step keys `calendar`/`invite`/`meeting` match `checklist-steps.ts`. `parseUrlLocale` returns `UrlLocale | null` — tests assert `toBeNull()`. ✓

**Known runtime risk (documented, not a placeholder):** the `date-fns`/`react-day-picker` locale `.code` field and `@leadcat/ui` TS resolution are the two most likely red-run surprises; both have an inline fallback instruction in Tasks 1–2.
