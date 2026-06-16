import { Badge, Button, Paw } from "@leadcat/ui"

import { CatStage } from "~/features/landing/components/cat-stage"
import { FloatingMotifs } from "~/features/landing/components/floating-motifs"
import {
  gsap,
  prefersCoarsePointer,
  useSceneMotion,
} from "~/features/landing/lib/motion"
import { useT } from "~/shared/i18n/context"

export function Hero() {
  const t = useT()

  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    const intro = gsap.timeline({
      defaults: { ease: "back.out(1.6)", duration: 0.8 },
    })
    intro
      .from(q("[data-hero-badge]"), { y: 18, autoAlpha: 0 })
      .from(
        q("[data-hero-line]"),
        { yPercent: 40, autoAlpha: 0, stagger: 0.12 },
        "-=0.4"
      )
      .from(q("[data-hero-copy]"), { y: 18, autoAlpha: 0 }, "-=0.4")
      .from(
        q("[data-hero-cta]"),
        { y: 14, autoAlpha: 0, stagger: 0.1 },
        "-=0.5"
      )
      .from(
        q("[data-hero-stage]"),
        { scale: 0.85, autoAlpha: 0, duration: 1, ease: "power3.out" },
        0.15
      )

    gsap.to(q("[data-hero-stage]"), {
      y: 10,
      duration: 3.2,
      ease: "sine.inOut",
      yoyo: true,
      repeat: -1,
    })

    gsap.to(q("[data-hero-motifs]"), {
      yPercent: -18,
      ease: "none",
      scrollTrigger: {
        trigger: root,
        start: "top top",
        end: "bottom top",
        scrub: true,
      },
    })

    if (prefersCoarsePointer()) {
      return
    }
    const motifs = gsap.utils.toArray<HTMLElement>("[data-parallax]", root)
    const setters = motifs.map((m) => ({
      x: gsap.quickTo(m, "x", { duration: 0.6, ease: "power3" }),
      y: gsap.quickTo(m, "y", { duration: 0.6, ease: "power3" }),
      depth: Number(m.dataset.parallax ?? 1),
    }))
    const onMove = (event: PointerEvent) => {
      const rect = root.getBoundingClientRect()
      const nx = (event.clientX - rect.left) / rect.width - 0.5
      const ny = (event.clientY - rect.top) / rect.height - 0.5
      for (const setter of setters) {
        setter.x(nx * 26 * setter.depth)
        setter.y(ny * 26 * setter.depth)
      }
    }
    root.addEventListener("pointermove", onMove)
    return () => root.removeEventListener("pointermove", onMove)
  })

  return (
    <section
      ref={scope}
      className="relative overflow-x-clip px-6 pt-8 pb-16 md:pt-16 md:pb-20"
    >
      <FloatingMotifs />
      <div className="relative mx-auto grid max-w-6xl grid-cols-1 items-center gap-8 md:grid-cols-2 md:gap-4">
        <div className="text-center md:text-left">
          <div data-hero-badge className="inline-block">
            <Badge tone="sunny" className="mb-5">
              <Paw className="size-3.5" />
              {t("hero.badge")}
            </Badge>
          </div>
          <h1 className="text-4xl leading-tight font-bold text-kitty-800 sm:text-5xl lg:text-6xl">
            <span data-hero-line className="block">
              {t("hero.titleLine1")}
            </span>
            <span data-hero-line className="block">
              <span className="relative whitespace-nowrap text-coral-500">
                {t("hero.titleHighlight")}
                <span
                  className="absolute -bottom-2 left-0 h-3 w-full rounded-full bg-sunny-300/70"
                  aria-hidden="true"
                />
              </span>{" "}
              <Paw className="inline-block size-8 align-middle text-coral-400 sm:size-10" />
            </span>
          </h1>
          <p
            data-hero-copy
            className="mx-auto mt-6 max-w-md text-lg text-kitty-600 md:mx-0"
          >
            {t("hero.copy")}
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-4 md:justify-start">
            <span data-hero-cta className="inline-block">
              <Button size="lg">{t("hero.ctaPrimary")}</Button>
            </span>
            <span data-hero-cta className="inline-block">
              <Button variant="secondary" size="lg" asChild>
                <a href="#how">{t("hero.ctaSecondary")}</a>
              </Button>
            </span>
          </div>
          <p
            data-hero-copy
            className="mt-5 text-sm font-semibold text-kitty-400"
          >
            {t("hero.footnote")}
          </p>
        </div>

        <div
          data-hero-stage
          className="relative mx-auto w-full max-w-xl overflow-visible md:max-w-lg"
        >
          <div className="relative h-[18rem] w-full sm:h-[22rem] md:h-[26rem]">
            <CatStage />
          </div>
        </div>
      </div>
    </section>
  )
}
