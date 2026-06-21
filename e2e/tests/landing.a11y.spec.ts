import { test } from "@playwright/test"
import { expectNoA11yViolations } from "../helpers/a11y"

test.describe("@a11y", () => {
  test("landing home (ru) has no critical/serious a11y violations", async ({ page }) => {
    await page.goto("http://localhost:8091/")
    await expectNoA11yViolations(page, "landing /")
  })

  test("landing home (en) has no critical/serious a11y violations", async ({ page }) => {
    await page.goto("http://localhost:8091/en")
    await expectNoA11yViolations(page, "landing /en")
  })
})
