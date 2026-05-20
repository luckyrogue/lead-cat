import { createRootRoute, Outlet, useRouterState } from "@tanstack/react-router";
import { AuthGate } from "@/features/auth/auth-gate";
import { LinkTelegramBanner } from "@/features/auth-link-telegram/link-telegram-banner";
import { CatShell } from "@/widgets/cat-shell/cat-shell";

function RootLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const onLogin = pathname === "/login";

  return (
    <AuthGate>
      {onLogin ? (
        <Outlet />
      ) : (
        <CatShell>
          <LinkTelegramBanner />
          <Outlet />
        </CatShell>
      )}
    </AuthGate>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
