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

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"
import { useT } from "~/shared/i18n/context"
import { useLandingDict } from "~/shared/i18n/use-landing-dict"

const featureIcons: LucideIcon[] = [CalendarDays, Bell, Heart, Star]

const featureTints = [
  "from-peach-100 to-cream-200",
  "from-sunny-100 to-cream-200",
  "from-coral-100 to-peach-100",
  "from-kitty-100 to-cream-200",
]

export function Features() {
  const t = useT()
  const dict = useLandingDict()

  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    gsap.from(q("[data-features-head]"), {
      y: 28,
      autoAlpha: 0,
      duration: 0.8,
      stagger: 0.12,
      ease: "power3.out",
      scrollTrigger: { trigger: root, start: "top 75%" },
    })

    gsap.from(q("[data-feature-card]"), {
      y: 40,
      autoAlpha: 0,
      duration: 0.7,
      stagger: 0.12,
      ease: "back.out(1.5)",
      scrollTrigger: { trigger: q("[data-features-grid]"), start: "top 80%" },
    })
  })

  return (
    <section ref={scope} id="features" className="relative px-6 py-20">
      <div className="mx-auto max-w-6xl">
        <div className="mb-12 text-center">
          <div data-features-head className="mb-4 flex justify-center">
            <CatFace className="animate-float size-16" />
          </div>
          <h2
            data-features-head
            className="text-3xl font-bold text-kitty-800 sm:text-4xl"
          >
            {t("features.title")}
          </h2>
          <p
            data-features-head
            className="mx-auto mt-3 max-w-xl text-kitty-600"
          >
            {t("features.subtitle")}
          </p>
        </div>

        <div
          data-features-grid
          className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4"
        >
          {dict.features.items.map((feature, index) => {
            const Icon = featureIcons[index] ?? CalendarDays
            const tint = featureTints[index] ?? featureTints[0]

            return (
              <Card
                key={feature.title}
                data-feature-card
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
                    {feature.title}
                  </h3>
                  <p className="mt-2 text-sm leading-relaxed text-kitty-600">
                    {feature.body}
                  </p>
                </div>
              </Card>
            )
          })}
        </div>
      </div>
    </section>
  )
}
