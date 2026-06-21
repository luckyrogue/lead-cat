import type { Page } from "@playwright/test"
import { getLatestMagicLink } from "./mailpit"

const ADMIN_LOCALE_KEY = "leadcat_admin_locale"

/** Pin English so getByLabel/getByRole matchers stay stable in CI. */
async function pinEnglishLocale(page: Page): Promise<void> {
  await page.addInitScript((key) => {
    window.localStorage.setItem(key, "en")
  }, ADMIN_LOCALE_KEY)
}

export async function loginViaMagicLink(page: Page, email: string): Promise<void> {
  await pinEnglishLocale(page)
  await page.goto("/login")
  await page.getByLabel("Email").fill(email)
  await page.getByRole("button", { name: "Send magic link" }).click()
  await page.getByText("Check your inbox").waitFor({ timeout: 15_000 })

  const link = await getLatestMagicLink(email)
  await page.goto(link)

  await page.waitForURL(
    (url) =>
      url.pathname === "/onboarding" ||
      url.pathname === "/" ||
      (!url.pathname.startsWith("/login") && !url.pathname.startsWith("/auth/magic")),
    { timeout: 45_000 }
  )
}
