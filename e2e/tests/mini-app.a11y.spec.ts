import { type Page, test } from "@playwright/test"

import { expectNoA11yViolations } from "../helpers/a11y"
import { stubTelegramWebApp } from "../helpers/tma"

const MINI_APP = "http://localhost:8092"

// TMA auth seam: the mini-app calls POST /api/auth/miniapp with
// window.Telegram.WebApp.initData. In the e2e stack, AUTH_DEV_MODE=true, but
// the dev-bypass loopback check fails because the browser's request reaches
// the backend via the nginx proxy (non-loopback IP). We therefore intercept
// the auth exchange at the Playwright network layer, returning a syntactically
// valid stub session, then mock the downstream miniapp API calls so the
// authenticated UI shells render. This gives real DOM / a11y coverage of every
// authenticated route without requiring production code changes or a seeded
// database user.

const STUB_USER = {
  telegram_id: 80000001,
  name: "A11y Tester",
  email: "a11y-tester@e2e.test",
  role: "user",
}

// setupMiniAppMocks wires route intercepts so that:
//   1. POST /api/auth/miniapp  → succeeds with a stub token + user
//   2. GET  /api/miniapp/me    → returns the stub user
//   3. GET  /api/miniapp/meetings* → returns an empty meetings list
//   4. GET  /api/miniapp/settings  → returns minimal settings
//   5. GET  /api/miniapp/calendar/* → returns empty connections
// The mocked token is never sent to a real backend; all guarded routes are
// intercepted, so subsequent 401s cannot cascade.
async function setupMiniAppMocks(page: Page): Promise<void> {
  await page.route("**/api/auth/miniapp", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ token: "e2e-stub-token", user: STUB_USER }),
    })
  })

  await page.route("**/api/miniapp/me", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(STUB_USER),
    })
  })

  await page.route("**/api/miniapp/meetings**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ meetings: [] }),
    })
  })

  await page.route("**/api/miniapp/settings", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ timezone: "UTC", language: "en", reminder_minutes: "15" }),
    })
  })

  await page.route("**/api/miniapp/calendar/**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ connections: [] }),
    })
  })
}

test.describe("@a11y", () => {
  test("mini-app authed pages have no critical/serious a11y violations", async ({ page }) => {
  const tgId = STUB_USER.telegram_id
  await stubTelegramWebApp(page, tgId)
  await setupMiniAppMocks(page)

  // Home page ( / )
  await page.goto(`${MINI_APP}/`)
  await page.waitForLoadState("networkidle")
  await expectNoA11yViolations(page, "mini-app /")

  // Meetings list ( /meetings )
  await page.goto(`${MINI_APP}/meetings`)
  await page.waitForLoadState("networkidle")
  await expectNoA11yViolations(page, "mini-app /meetings")

  // Profile ( /profile )
  await page.goto(`${MINI_APP}/profile`)
  await page.waitForLoadState("networkidle")
  await expectNoA11yViolations(page, "mini-app /profile")
  })
})
