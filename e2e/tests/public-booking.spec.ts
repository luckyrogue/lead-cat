import { expect, test } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"

test("flow 2: public /book/:slug booking -> confirmation", async ({ page, browser }) => {
  const email = `owner-${Date.now()}@e2e.test`
  const orgName = `E2E Org ${Date.now()}`
  let slug: string

  await test.step("magic-link login", async () => {
    await loginViaMagicLink(page, email)
    await expect(page.getByLabel("Organization name")).toBeVisible()
  })

  await test.step("create org and reach dashboard", async () => {
    await page.getByLabel("Organization name").fill(orgName)
    await page.getByRole("button", { name: "Create organization" }).click()
    await page.waitForURL((url) => url.pathname === "/")
  })

  await test.step("resolve org id and create event type via API", async () => {
    const orgsRes = await page.request.get("/api/orgs")
    expect(orgsRes.ok(), `GET /api/orgs status: ${orgsRes.status()}`).toBeTruthy()
    const orgsBody = await orgsRes.json()
    const orgId: string = orgsBody.organizations[0].id

    const cookies = await page.context().cookies()
    const csrf = cookies.find((c) => c.name === "lc_csrf")?.value ?? ""

    const etRes = await page.request.post("/api/booking/event-types", {
      headers: {
        "X-Org-Id": orgId,
        "X-CSRF-Token": csrf,
      },
      data: {
        title: "E2E Intro",
        description: "",
        duration_mins: 30,
        timezone: "Asia/Almaty",
        avail_weekdays: [1, 2, 3, 4, 5],
        avail_start_minute: 540,
        avail_end_minute: 1020,
        active: true,
      },
    })
    expect(
      etRes.ok(),
      `POST /api/booking/event-types status: ${etRes.status()} body: ${await etRes.text()}`
    ).toBeTruthy()
    const et = await etRes.json()
    slug = et.slug
    expect(slug, "event type slug must be present").toBeTruthy()
  })

  await test.step("visitor books a slot on the public page", async () => {
    const ctx = await browser.newContext()
    const visitor = await ctx.newPage()

    try {
      await visitor.goto(`/book/${slug}`)

      await expect(visitor.getByText("E2E Intro")).toBeVisible()

      const dayCard = visitor.locator('[data-slot="card"]').filter({ hasText: "Select a day" })
      const firstDayButton = dayCard.locator('[data-slot="card-content"] button[type="button"]').first()
      await expect(firstDayButton).toBeVisible()
      await firstDayButton.click()

      const allCards = visitor.locator('[data-slot="card"]')
      const slotCard = allCards.nth(2)
      const firstSlotButton = slotCard.locator('button[type="button"]').first()
      await expect(firstSlotButton).toBeVisible()
      await firstSlotButton.click()

      await expect(visitor.getByRole("button", { name: "Continue" })).toBeVisible()
      await visitor.getByRole("button", { name: "Continue" }).click()

      await expect(visitor.getByLabel("Name")).toBeVisible()
      await visitor.getByLabel("Name").fill("E2E Visitor")
      await visitor.getByLabel("Email").fill("visitor@e2e.test")

      await visitor.getByRole("button", { name: "Confirm booking" }).click()

      await expect(visitor.getByText("You're booked!")).toBeVisible()
      await expect(visitor.getByRole("link", { name: "Join Google Meet" })).toBeVisible()
    } finally {
      await ctx.close()
    }
  })
})
