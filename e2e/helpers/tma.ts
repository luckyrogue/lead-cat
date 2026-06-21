import type { Page } from "@playwright/test"

// stubTelegramWebApp injects a fake Telegram WebApp object before the mini-app
// loads, so getInitData() returns `telegramID`. The e2e backend runs with
// AUTH_DEV_MODE=true (APP_ENV=development), where POST /api/auth/miniapp accepts a
// plain int64 telegram id as init_data and maps it to a dev user. Test-only; no
// production code path enables this.
export async function stubTelegramWebApp(page: Page, telegramID: number): Promise<void> {
  await page.addInitScript((id: number) => {
    ;(window as unknown as { Telegram: unknown }).Telegram = {
      WebApp: {
        initData: String(id),
        initDataUnsafe: { user: { id, first_name: "A11y" } },
        ready() {},
        expand() {},
        colorScheme: "light",
        openLink() {},
      },
    }
  }, telegramID)
}
