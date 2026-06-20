import { expect, test } from "@playwright/test"
import type { Locator, Page } from "@playwright/test"
import { loginViaMagicLink } from "../helpers/auth"

async function pickTimeInPopover(page: Page, triggerLocator: Locator, hour: string, minute: string) {
  await triggerLocator.click()
  const popover = page.locator("[data-radix-popper-content-wrapper]").filter({ hasText: "Hour" })
  await expect(popover).toBeVisible()
  await popover.getByRole("combobox").nth(0).click()
  await page.getByRole("option", { name: hour, exact: true }).click()
  await popover.getByRole("combobox").nth(1).click()
  await page.getByRole("option", { name: minute, exact: true }).click()
  await page.keyboard.press("Escape")
}

test("flow 1: magic-link auth → create org → create meeting", async ({ page }) => {
  const email = `owner-${Date.now()}@e2e.test`
  const orgName = `E2E Org ${Date.now()}`
  const meetingType = `Standup ${Date.now()}`

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

  await test.step("open New meeting dialog, fill form, and verify row in table", async () => {
    await page.goto("/meetings")
    await expect(page.getByRole("button", { name: "New meeting" })).toBeVisible()
    await page.getByRole("button", { name: "New meeting" }).click()

    const dialog = page.getByRole("dialog", { name: "New meeting" })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder("Engineering").fill("Engineering")
    await dialog.getByPlaceholder("Standup").fill(meetingType)
    await dialog.getByPlaceholder("Host name").fill("Alice")

    const startTrigger = dialog.getByRole("button").filter({ hasText: /10:00/ }).first()
    await pickTimeInPopover(page, startTrigger, "10", "00")

    const endTrigger = dialog.getByRole("button").filter({ hasText: /11:00/ }).first()
    await pickTimeInPopover(page, endTrigger, "11", "00")

    await dialog.getByPlaceholder("alice@company.com, bob@company.com").fill("participant@e2e.test")

    await dialog.getByRole("button", { name: "Create meeting" }).click()

    await expect(page.getByRole("cell", { name: meetingType })).toBeVisible()
  })
})
