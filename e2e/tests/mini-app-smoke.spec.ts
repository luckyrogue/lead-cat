import { expect, test } from "@playwright/test"

test("mini-app shell loads", async ({ page }) => {
  const res = await page.goto("http://localhost:8092/")
  expect(res?.ok()).toBeTruthy()
})
