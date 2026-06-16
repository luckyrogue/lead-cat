import { Badge } from "@leadcat/ui"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"

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

// `to`/`suffix` drive the count-up; the JSX renders the final value so SSR and
// reduced-motion users always see the real number.
const stats = [
  { to: 3, suffix: " sec", label: "to book a meeting" },
  { to: 0, suffix: "", label: "double-bookings" },
  { to: 100, suffix: "%", label: "more purring" },
]

export function HowItWorks() {
  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    gsap.from(q("[data-how-head]"), {
      y: 24,
      autoAlpha: 0,
      duration: 0.8,
      ease: "power3.out",
      scrollTrigger: { trigger: root, start: "top 72%" },
    })

    gsap.from(q("[data-how-step]"), {
      y: 36,
      autoAlpha: 0,
      duration: 0.7,
      stagger: 0.15,
      ease: "back.out(1.6)",
      scrollTrigger: { trigger: root, start: "top 65%" },
    })

    // Count each stat up from zero when the stat row scrolls into view.
    const counters = q("[data-countup]") as unknown as HTMLElement[]
    counters.forEach((el) => {
      const to = Number(el.dataset.to ?? 0)
      const suffix = el.dataset.suffix ?? ""
      const obj = { v: 0 }
      el.textContent = `0${suffix}`
      gsap.to(obj, {
        v: to,
        duration: 1.4,
        ease: "power2.out",
        scrollTrigger: { trigger: q("[data-how-stats]"), start: "top 85%" },
        onUpdate: () => {
          el.textContent = `${Math.round(obj.v)}${suffix}`
        },
      })
    })
  })

  return (
    <section ref={scope} id="how" className="relative px-6 py-12">
      <div className="mx-auto max-w-5xl rounded-[calc(var(--radius)*2)] bg-gradient-to-br from-coral-400 to-peach-400 p-8 shadow-[var(--shadow-soft)] sm:p-12">
        <div className="mb-10 text-center">
          <div data-how-head className="inline-block">
            <Badge className="border-white/40 bg-white/20 text-white">
              How it works
            </Badge>
          </div>
          <h2
            data-how-head
            className="mt-4 text-3xl font-bold text-white sm:text-4xl"
          >
            Three little steps, zero headaches
          </h2>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          {steps.map((step) => (
            <div
              key={step.n}
              data-how-step
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

        <div
          data-how-stats
          className="mt-10 grid grid-cols-3 gap-4 border-t border-white/30 pt-8 text-center text-white"
        >
          {stats.map((stat) => (
            <div key={stat.label}>
              <div className="text-3xl font-bold sm:text-4xl">
                <span data-countup data-to={stat.to} data-suffix={stat.suffix}>
                  {stat.to}
                  {stat.suffix}
                </span>
              </div>
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
