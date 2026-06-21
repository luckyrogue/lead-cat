import { test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"
import { expectNoA11yViolations } from "../helpers/a11y"

test.describe("@a11y", () => {
  test("admin authed pages have no critical/serious a11y violations", async ({ page }) => {
  const email = `a11y-admin-${Date.now()}@e2e.test`

  await loginViaMagicLink(page, email)
  // Onboarding (pre-org) page.
  await expectNoA11yViolations(page, "admin /onboarding")

  await page.getByLabel("Organization name").fill(`A11y Admin ${Date.now()}`)
  await page.getByRole("button", { name: "Create organization" }).click()
  await page.waitForURL((url) => url.pathname === "/")
  await expectNoA11yViolations(page, "admin / (dashboard)")

  await page.goto("/meetings")
  await page.waitForURL(/\/meetings/)
  await expectNoA11yViolations(page, "admin /meetings")

  // Booking config page (route that lists/creates event types).
  await page.goto("/booking")
  await expectNoA11yViolations(page, "admin /booking")
  })
})
