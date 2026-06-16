import { Badge, CalendarDays, CatFace, Check, Clock, Paw } from "@leadcat/ui"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"
import { useT } from "~/shared/i18n/context"
import { useLandingDict } from "~/shared/i18n/use-landing-dict"

const meetingTints = [
  "bg-peach-100 text-coral-500",
  "bg-sunny-100 text-sunny-400",
  "bg-coral-100 text-coral-500",
]

export function Showcase() {
  const t = useT()
  const dict = useLandingDict()

  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    gsap.from(q("[data-show-copy]"), {
      x: -40,
      autoAlpha: 0,
      duration: 0.9,
      stagger: 0.12,
      ease: "power3.out",
      scrollTrigger: { trigger: root, start: "top 70%" },
    })

    gsap.fromTo(
      q("[data-show-device]"),
      {
        clipPath: "inset(0 0 100% 0 round 2rem)",
        autoAlpha: 0,
        y: 36,
        rotateZ: -3,
      },
      {
        clipPath: "inset(0 0 0% 0 round 2rem)",
        autoAlpha: 1,
        y: 0,
        rotateZ: 0,
        duration: 1.1,
        ease: "power3.out",
        scrollTrigger: { trigger: root, start: "top 68%" },
      }
    )

    gsap.from(q("[data-show-row]"), {
      y: 16,
      autoAlpha: 0,
      stagger: 0.12,
      delay: 0.35,
      ease: "back.out(1.7)",
      scrollTrigger: { trigger: root, start: "top 68%" },
    })
  })

  return (
    <section
      ref={scope}
      id="showcase"
      className="relative overflow-hidden px-6 py-20"
    >
      <div className="mx-auto grid max-w-6xl grid-cols-1 items-center gap-12 md:grid-cols-2">
        <div className="text-center md:text-left">
          <div data-show-copy className="inline-block">
            <Badge tone="sunny" className="mb-5">
              <Paw className="size-3.5" />
              {t("showcase.badge")}
            </Badge>
          </div>
          <h2
            data-show-copy
            className="text-3xl font-bold text-kitty-800 sm:text-4xl"
          >
            {t("showcase.title")}
          </h2>
          <p
            data-show-copy
            className="mx-auto mt-4 max-w-md text-kitty-600 md:mx-0"
          >
            {t("showcase.copy")}
          </p>
          <ul className="mt-7 flex flex-col gap-3 text-left">
            {dict.showcase.perks.map((perk) => (
              <li
                key={perk}
                data-show-copy
                className="flex items-start gap-3 text-sm font-medium text-kitty-600"
              >
                <span className="mt-0.5 grid size-6 shrink-0 place-items-center rounded-full bg-coral-100 text-coral-500">
                  <Check className="size-3.5" />
                </span>
                {perk}
              </li>
            ))}
          </ul>
        </div>

        <div className="relative mx-auto w-full max-w-xs">
          <div
            className="absolute -inset-6 rounded-[3rem] bg-gradient-to-br from-peach-200/60 to-sunny-200/50 blur-2xl"
            aria-hidden="true"
          />
          <div
            data-show-device
            className="surface-card relative rounded-[2rem] p-5"
          >
            <div className="mb-5 flex items-center justify-between">
              <div className="flex items-center gap-2 font-bold text-kitty-800">
                <CatFace className="size-7" />
                {t("showcase.today")}
              </div>
              <span className="flex items-center gap-1 text-xs font-semibold text-kitty-400">
                <CalendarDays className="size-4" />
                {t("showcase.dateLabel")}
              </span>
            </div>

            <div className="flex flex-col gap-3">
              {dict.showcase.meetings.map((meeting, index) => (
                <div
                  key={meeting.title}
                  data-show-row
                  className="flex items-center gap-3 rounded-2xl bg-cream-50 p-3"
                >
                  <span
                    className={`grid size-10 shrink-0 place-items-center rounded-xl text-xs font-bold ${meetingTints[index] ?? meetingTints[0]}`}
                  >
                    <Clock className="size-4" />
                  </span>
                  <div className="min-w-0">
                    <div className="text-sm font-bold text-kitty-800">
                      {meeting.title}
                    </div>
                    <div className="text-xs font-medium text-kitty-400">
                      {meeting.time}
                    </div>
                  </div>
                  <Paw className="ml-auto size-4 text-coral-300" />
                </div>
              ))}
            </div>

            <div className="mt-5 grid place-items-center rounded-2xl bg-gradient-to-br from-coral-400 to-peach-400 py-3 text-sm font-bold text-white shadow-[var(--shadow-soft)]">
              {t("showcase.bookedIn")}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
