import type { Page } from "@playwright/test"
import { getLatestMagicLink } from "./mailpit"

export async function loginViaMagicLink(page: Page, email: string): Promise<void> {
  await page.goto("/login")
  await page.getByLabel("Email").fill(email)
  await page.getByRole("button", { name: "Send magic link" }).click()
  const link = await getLatestMagicLink(email)
  await page.goto(link)
  await page.waitForURL((url) => !url.pathname.startsWith("/login"))
}
