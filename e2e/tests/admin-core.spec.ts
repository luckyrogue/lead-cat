import { expect, test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"

test("flow 1: magic-link auth -> create org -> reach dashboard + meetings", async ({ page }) => {
  const email = `owner-${Date.now()}@e2e.test`
  const orgName = `E2E Org ${Date.now()}`

  await test.step("magic-link login lands on /onboarding", async () => {
    await loginViaMagicLink(page, email)
    await expect(page.getByLabel("Organization name")).toBeVisible()
    await expect(page).toHaveURL(/\/onboarding/)
  })

  await test.step("fill org name and reach dashboard", async () => {
    await page.getByLabel("Organization name").fill(orgName)
    await page.getByRole("button", { name: "Create organization" }).click()
    await page.waitForURL((url) => url.pathname === "/")
  })

  await test.step("authed app routes to the meetings page", async () => {
    await page.goto("/meetings")
    await expect(page).toHaveURL(/\/meetings/)
    await expect(
      page.getByRole("button", { name: /New meeting|Новая встреча|Жаңа кездесу/ })
    ).toBeVisible()
  })
})
