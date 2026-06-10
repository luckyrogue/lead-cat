import {
  Bell,
  CalendarDays,
  CatFace,
  Card,
  Heart,
  type LucideIcon,
  Paw,
  Star,
} from "@leadcat/ui"

type Feature = {
  icon: LucideIcon
  title: string
  body: string
  tint: string
}

const features: Feature[] = [
  {
    icon: CalendarDays,
    title: "One calm calendar view",
    body: "Lead Cat merges every calendar into one tidy availability map - Google, work, personal. See open paws at a glance.",
    tint: "from-peach-100 to-cream-200",
  },
  {
    icon: Bell,
    title: "Reminders that don't nag",
    body: "Gentle, smart nudges right when they matter. Your team shows up on time without a single grumpy meow.",
    tint: "from-sunny-100 to-cream-200",
  },
  {
    icon: Heart,
    title: "Team scheduling, sorted",
    body: "Pick a group, and Lead Cat pounces on the slot that works for everyone. Round-robin and pooled hosts included.",
    tint: "from-coral-100 to-peach-100",
  },
  {
    icon: Star,
    title: "Lives where you work",
    body: "Native Google Calendar sync and a cozy Telegram Mini App. Book, reschedule and confirm without leaving chat.",
    tint: "from-kitty-100 to-cream-200",
  },
]

export function Features() {
  return (
    <section id="features" className="relative px-6 py-20">
      <div className="mx-auto max-w-6xl">
        <div className="mb-12 text-center">
          <div className="mb-4 flex justify-center">
            <CatFace className="size-16 animate-float" />
          </div>
          <h2 className="text-3xl font-bold text-kitty-800 sm:text-4xl">
            Everything you need to herd the cats
          </h2>
          <p className="mx-auto mt-3 max-w-xl text-kitty-600">
            Powerful scheduling under the hood, wrapped in something genuinely
            delightful to use.
          </p>
        </div>

        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {features.map(({ icon: Icon, title, body, tint }) => (
            <Card
              key={title}
              className="group h-full transition-transform duration-300 [transition-timing-function:var(--ease-spring)] hover:-translate-y-2 hover:rotate-1"
            >
              <div className="px-5">
                <div
                  className={`mb-5 inline-grid size-16 place-items-center rounded-2xl bg-gradient-to-br ${tint} text-coral-500 transition-transform duration-300 [transition-timing-function:var(--ease-spring)] group-hover:scale-110 group-hover:-rotate-6`}
                >
                  <Icon className="size-8" />
                </div>
                <h3 className="flex items-center gap-2 text-lg font-bold text-kitty-800">
                  <Paw className="size-4 text-coral-400" />
                  {title}
                </h3>
                <p className="mt-2 text-sm leading-relaxed text-kitty-600">
                  {body}
                </p>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </section>
  )
}
