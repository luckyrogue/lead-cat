import { test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"
import { expectNoA11yViolations } from "../helpers/a11y"

test("public booking page has no critical/serious a11y violations", async ({ page }) => {
  const email = `a11y-${Date.now()}@e2e.test`
  await loginViaMagicLink(page, email)
  await page.getByLabel("Organization name").fill(`A11y Org ${Date.now()}`)
  await page.getByRole("button", { name: "Create organization" }).click()
  await page.waitForURL((url) => url.pathname === "/")

  const orgsRes = await page.request.get("/api/orgs")
  const orgId: string = (await orgsRes.json()).organizations[0].id
  const csrf =
    (await page.context().cookies()).find((c) => c.name === "lc_csrf")?.value ?? ""
  const etRes = await page.request.post("/api/booking/event-types", {
    headers: { "X-Org-Id": orgId, "X-CSRF-Token": csrf },
    data: {
      title: "A11y Intro",
      description: "",
      duration_mins: 30,
      timezone: "Asia/Almaty",
      avail_weekdays: [1, 2, 3, 4, 5],
      avail_start_minute: 540,
      avail_end_minute: 1020,
      active: true,
    },
  })
  const slug: string = (await etRes.json()).slug

  await page.goto(`/book/${slug}`)
  await expectNoA11yViolations(page, `/book/${slug}`)
})
