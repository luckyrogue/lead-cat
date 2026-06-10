import {
  ArrowUpRight,
  Button,
  CalendarDays,
  Card,
  CardContent,
  Users,
  Video,
} from "@leadcat/ui"

const stats = [
  {
    key: "meetings",
    title: "Meetings",
    total: 128,
    secondary: "Scheduled across all operators",
    icon: Video,
  },
  {
    key: "operators",
    title: "Operators",
    total: 12,
    secondary: "9 active / 3 invited",
    icon: Users,
  },
  {
    key: "today",
    title: "Today",
    total: 7,
    secondary: "Upcoming in the next 24 hours",
    icon: CalendarDays,
  },
]

export function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1">
          <p className="text-[0.68rem] font-semibold tracking-[0.16em] text-muted-foreground uppercase">
            Lead Cat Admin
          </p>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">
            Overview
          </h1>
          <p className="max-w-xl text-sm text-muted-foreground">
            A calm snapshot of meetings and operators across the workspace.
          </p>
        </div>
        <Button>
          New meeting
          <ArrowUpRight className="size-4" />
        </Button>
      </header>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <Card key={stat.key}>
              <CardContent className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">
                    {stat.title}
                  </p>
                  <p className="mt-2 text-3xl font-semibold tracking-tight">
                    {stat.total}
                  </p>
                  <p className="mt-2 text-sm text-muted-foreground">
                    {stat.secondary}
                  </p>
                </div>
                <span className="rounded-[calc(var(--radius)*0.7)] border border-border/70 bg-background/70 p-2 text-primary">
                  <Icon className="size-5" />
                </span>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
