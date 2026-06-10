import {
  Bell,
  Button,
  CalendarDays,
  Card,
  CardContent,
  CatFace,
  Clock,
  type LucideIcon,
} from "@leadcat/ui"

type Upcoming = {
  icon: LucideIcon
  title: string
  when: string
}

const upcoming: Upcoming[] = [
  { icon: CalendarDays, title: "Weekly sync", when: "Today, 14:00" },
  { icon: Clock, title: "Design review", when: "Tomorrow, 10:30" },
  { icon: Bell, title: "1:1 with Aru", when: "Fri, 16:00" },
]

export function HomePage() {
  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center gap-3">
        <CatFace className="size-12 animate-float" />
        <div>
          <p className="text-sm font-semibold text-muted-foreground">
            Lead Cat
          </p>
          <h1 className="text-xl font-bold text-kitty-800">Your meetings</h1>
        </div>
      </header>

      <Button size="lg" className="w-full">
        Book a meeting
      </Button>

      <div className="flex flex-col gap-3">
        {upcoming.map((item) => {
          const Icon = item.icon
          return (
            <Card key={item.title}>
              <CardContent className="flex items-center gap-3">
                <span className="rounded-[calc(var(--radius)*0.7)] border border-border/70 bg-background/80 p-2 text-primary">
                  <Icon className="size-5" />
                </span>
                <div className="min-w-0">
                  <p className="font-semibold text-foreground">{item.title}</p>
                  <p className="text-sm text-muted-foreground">{item.when}</p>
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
