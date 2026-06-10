import { Badge } from "@leadcat/ui"

const steps = [
  {
    n: "1",
    title: "Connect your calendars",
    body: "Link Google in two taps. Lead Cat learns when you're free.",
  },
  {
    n: "2",
    title: "Share your kitty link",
    body: "Drop one link in chat. Guests pick a slot that just works.",
  },
  {
    n: "3",
    title: "Show up and relax",
    body: "Auto-invites, reminders and reschedules. You do nothing.",
  },
]

const stats = [
  { value: "3 sec", label: "to book a meeting" },
  { value: "0", label: "double-bookings" },
  { value: "100%", label: "more purring" },
]

export function HowItWorks() {
  return (
    <section id="how" className="relative px-6 py-12">
      <div className="mx-auto max-w-5xl rounded-[calc(var(--radius)*2)] bg-gradient-to-br from-coral-400 to-peach-400 p-8 shadow-[var(--shadow-soft)] sm:p-12">
        <div className="mb-10 text-center">
          <Badge className="border-white/40 bg-white/20 text-white">
            How it works
          </Badge>
          <h2 className="mt-4 text-3xl font-bold text-white sm:text-4xl">
            Three little steps, zero headaches
          </h2>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          {steps.map((step) => (
            <div
              key={step.n}
              className="rounded-[calc(var(--radius)*1.6)] bg-white/90 p-6 text-center backdrop-blur"
            >
              <div className="mx-auto mb-4 grid size-12 place-items-center rounded-full bg-sunny-400 text-xl font-bold text-kitty-800 shadow-[0_10px_24px_-12px_oklch(0.85_0.15_88_/_0.8)]">
                {step.n}
              </div>
              <h3 className="text-lg font-bold text-kitty-800">{step.title}</h3>
              <p className="mt-2 text-sm text-kitty-600">{step.body}</p>
            </div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-3 gap-4 border-t border-white/30 pt-8 text-center text-white">
          {stats.map((stat) => (
            <div key={stat.label}>
              <div className="text-3xl font-bold sm:text-4xl">{stat.value}</div>
              <div className="mt-1 text-xs font-semibold tracking-wider text-white/80 uppercase">
                {stat.label}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
