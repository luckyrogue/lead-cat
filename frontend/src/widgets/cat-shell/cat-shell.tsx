import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { toast } from "sonner";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id";

const nav = [
  { to: "/workspaces", label: "Логова", needsWorkspace: false },
  { to: "/dashboard", label: "Дашборд", needsWorkspace: true },
  { to: "/scenarios", label: "Сценарии", needsWorkspace: true },
  { to: "/team", label: "Котики", needsWorkspace: true },
  { to: "/integrations", label: "Интеграции", needsWorkspace: true },
  { to: "/chat-link", label: "Привязка", needsWorkspace: true },
] as const;

export function CatShell({ children }: { children: ReactNode }) {
  const workspaceId = useWorkspaceId();

  return (
    <div className="flex min-h-screen flex-col">
      <header className="flex items-center gap-3 border-b border-border bg-card/80 px-4 py-3 backdrop-blur">
        <span className="text-2xl" aria-hidden>
          🐈
        </span>
        <div>
          <h1 className="text-lg font-bold text-primary">Lead Cat</h1>
          <p className="text-xs text-muted-foreground">мы любим котиков</p>
        </div>
      </header>
      <nav className="flex gap-1 overflow-x-auto px-2 py-2">
        {nav.map((item) => {
          const needsWorkspace = item.needsWorkspace;
          const blocked = needsWorkspace && !workspaceId;
          const linkClass = cn(
            buttonVariants({ variant: "ghost", size: "sm" }),
            "rounded-full whitespace-nowrap",
            blocked && "cursor-not-allowed opacity-50"
          );

          return (
            <Link
              key={item.to}
              to={item.to}
              search={needsWorkspace ? { workspaceId } : undefined}
              className={linkClass}
              activeProps={{
                className: cn(
                  buttonVariants({ variant: "default", size: "sm" }),
                  "rounded-full"
                ),
              }}
              onClick={(e) => {
                if (blocked) {
                  e.preventDefault();
                  toast.info("Сначала открой логово из списка на вкладке «Логова»");
                }
              }}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
      <main className="mx-auto w-full max-w-2xl flex-1 p-4">{children}</main>
    </div>
  );
}
