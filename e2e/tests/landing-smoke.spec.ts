import { expect, test } from "@playwright/test"

test("landing SSR home responds", async ({ page }) => {
  const res = await page.goto("http://localhost:8091/")
  expect(res?.ok()).toBeTruthy()
})

test("landing locale route responds", async ({ page }) => {
  const res = await page.goto("http://localhost:8091/en")
  expect(res?.ok()).toBeTruthy()
})
