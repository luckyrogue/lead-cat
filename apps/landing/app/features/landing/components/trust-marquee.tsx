import { Paw } from "@leadcat/ui"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"

const items = [
  "Google Calendar sync",
  "Telegram Mini App",
  "Zero double-bookings",
  "Round-robin hosts",
  "Gentle reminders",
  "3-second booking",
  "Pooled availability",
  "Reschedules itself",
]

export function TrustMarquee() {
  const scope = useSceneMotion<HTMLElement>((root) => {
    const track = root.querySelector<HTMLElement>("[data-marquee-track]")
    if (!track) {
      return
    }
    // Two identical halves sit side by side; sliding one full half and looping
    // reads as an endless ribbon.
    gsap.to(track, {
      xPercent: -50,
      ease: "none",
      duration: 28,
      repeat: -1,
    })
  })

  return (
    <section
      ref={scope}
      aria-label="What Lead Cat does"
      className="relative overflow-hidden border-y border-border/60 bg-cream-50/60 py-5"
    >
      {/* Soft edge fades so the ribbon melts into the page. */}
      <div
        className="pointer-events-none absolute inset-y-0 left-0 z-10 w-24 bg-gradient-to-r from-cream-50 to-transparent"
        aria-hidden="true"
      />
      <div
        className="pointer-events-none absolute inset-y-0 right-0 z-10 w-24 bg-gradient-to-l from-cream-50 to-transparent"
        aria-hidden="true"
      />
      <div data-marquee-track className="flex w-max items-center">
        {[0, 1].map((half) => (
          <ul
            key={half}
            className="flex shrink-0 items-center gap-8 pr-8"
            aria-hidden={half === 1}
          >
            {items.map((item) => (
              <li
                key={item}
                className="flex items-center gap-3 text-sm font-semibold whitespace-nowrap text-kitty-600"
              >
                <Paw className="size-4 text-coral-300" />
                {item}
              </li>
            ))}
          </ul>
        ))}
      </div>
    </section>
  )
}
