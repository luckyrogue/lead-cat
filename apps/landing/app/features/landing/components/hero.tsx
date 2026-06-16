import { Badge, Button, Paw } from "@leadcat/ui"

import { CatStage } from "~/features/landing/components/cat-stage"
import { FloatingMotifs } from "~/features/landing/components/floating-motifs"
import {
  gsap,
  prefersCoarsePointer,
  useSceneMotion,
} from "~/features/landing/lib/motion"

export function Hero() {
  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    // Above the fold → playful entrance plays immediately on mount.
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

    // The whole motif layer drifts up as the hero scrolls away.
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

    // Pointer parallax on the individual paws — desktop only.
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
      className="relative overflow-hidden px-6 pt-8 pb-12 md:pt-16"
    >
      <FloatingMotifs />
      <div className="relative mx-auto grid max-w-6xl grid-cols-1 items-center gap-8 md:grid-cols-2 md:gap-4">
        <div className="text-center md:text-left">
          <div data-hero-badge className="inline-block">
            <Badge tone="sunny" className="mb-5">
              <Paw className="size-3.5" />
              Scheduling, but cuddly
            </Badge>
          </div>
          <h1 className="text-4xl leading-tight font-bold text-kitty-800 sm:text-5xl lg:text-6xl">
            <span data-hero-line className="block">
              Meetings your team will
            </span>
            <span data-hero-line className="block">
              <span className="relative whitespace-nowrap text-coral-500">
                actually love
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
            Lead Cat sniffs out the purrfect time across everyone&apos;s
            calendars - no more back-and-forth, no more double-booking. Just
            happy little meetings.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-4 md:justify-start">
            <span data-hero-cta className="inline-block">
              <Button size="lg">Get started - it&apos;s free</Button>
            </span>
            <span data-hero-cta className="inline-block">
              <Button variant="secondary" size="lg">
                See how it works
              </Button>
            </span>
          </div>
          <p
            data-hero-copy
            className="mt-5 text-sm font-semibold text-kitty-400"
          >
            Works with Google Calendar &amp; Telegram. No credit card, no claws
            out.
          </p>
        </div>

        <div
          data-hero-stage
          className="relative mx-auto aspect-square w-full max-w-lg"
        >
          <div
            className="absolute inset-x-6 top-10 bottom-6 rounded-full bg-gradient-to-b from-peach-100 to-cream-200"
            aria-hidden="true"
          />
          <div className="relative h-full w-full">
            <CatStage />
          </div>
        </div>
      </div>
    </section>
  )
}
