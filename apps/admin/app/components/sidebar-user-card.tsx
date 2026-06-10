import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Building2,
  Check,
  ChevronsUpDown,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
  LogOut,
} from "@leadcat/ui"
import { Link } from "react-router"

import { cn } from "~/shared/lib/cn"
import type { Me } from "~/shared/auth/types"
import { useActiveOrg } from "~/shared/auth/use-active-org"

function initials(email: string) {
  const name = email.split("@")[0] ?? ""
  const tokens = name.split(/[._-]+/).filter(Boolean)
  if (tokens.length === 0) {
    return "LC"
  }
  if (tokens.length === 1) {
    return tokens[0].slice(0, 2).toUpperCase()
  }
  return tokens
    .slice(0, 2)
    .map((token) => token[0]?.toUpperCase() ?? "")
    .join("")
}

type SidebarUserCardProps = {
  me: Me
}

export function SidebarUserCard({ me }: SidebarUserCardProps) {
  const { activeOrg, activeOrgId, selectOrg } = useActiveOrg(me.organizations)
  const { user, organizations } = me

  return (
    <div className="shrink-0 space-y-3 border-t border-border/70 p-3">
      {organizations.length > 0 ? (
        <DropdownMenu>
          <DropdownMenuTrigger className="flex w-full items-center gap-2.5 rounded-[calc(var(--radius)*1.05)] border border-border/70 bg-background/70 px-3 py-2.5 text-left transition-colors outline-none hover:border-border focus-visible:ring-4 focus-visible:ring-ring/30">
            <span className="grid size-8 shrink-0 place-items-center rounded-[calc(var(--radius)*0.7)] border border-primary/15 bg-primary/10 text-primary">
              <Building2 className="size-4" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium text-foreground">
                {activeOrg?.name ?? "Select organization"}
              </span>
              <span className="block text-xs text-muted-foreground">
                Organization
              </span>
            </span>
            <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            className="w-[15rem]"
            sideOffset={8}
          >
            <DropdownMenuLabel>Organizations</DropdownMenuLabel>
            {organizations.map((org) => (
              <DropdownMenuItem key={org.id} onSelect={() => selectOrg(org.id)}>
                <Building2 className="text-muted-foreground" />
                <span className="flex-1 truncate">{org.name}</span>
                <Check
                  className={cn(
                    "ml-auto size-4 text-primary",
                    org.id === activeOrgId ? "opacity-100" : "opacity-0"
                  )}
                />
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      ) : null}

      <div className="flex items-center gap-3 rounded-[calc(var(--radius)*1.05)] border border-border/70 bg-background/70 px-3 py-2.5">
        <Avatar className="size-9">
          {user.avatar_url ? (
            <AvatarImage src={user.avatar_url} alt={user.email} />
          ) : null}
          <AvatarFallback>{initials(user.email)}</AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-foreground">
            {user.email}
          </p>
          <p className="text-xs text-muted-foreground">{user.auth_method}</p>
        </div>
        <Link
          to="/logout"
          aria-label="Log out"
          className="rounded-[calc(var(--radius)*0.7)] p-2 text-muted-foreground transition-colors hover:bg-accent/70 hover:text-foreground"
        >
          <LogOut className="size-4" />
        </Link>
      </div>
    </div>
  )
}
