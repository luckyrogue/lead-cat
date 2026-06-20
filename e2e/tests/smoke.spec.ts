import { expect, test } from "@playwright/test"

test("login page renders", async ({ page }) => {
  await page.goto("/login")
  await expect(page.getByLabel("Email")).toBeVisible()
  await expect(page.getByRole("button", { name: "Send magic link" })).toBeVisible()
})
