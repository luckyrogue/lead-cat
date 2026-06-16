import { Button, CatFace, Paw } from "@leadcat/ui"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"

const headingWords = ["Ready", "to", "make", "meetings", "purr?"]

const driftingPaws = [
  { className: "left-[12%] bottom-[8%] size-8 text-coral-200", delay: "0s" },
  { className: "left-[28%] bottom-[24%] size-6 text-sunny-300", delay: "0.8s" },
  {
    className: "right-[24%] bottom-[18%] size-7 text-peach-300",
    delay: "1.3s",
  },
  {
    className: "right-[12%] bottom-[10%] size-9 text-coral-200",
    delay: "0.4s",
  },
]

export function Footer() {
  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

    // Heading lights up word by word as it enters.
    gsap.from(q("[data-footer-word]"), {
      autoAlpha: 0.18,
      y: 12,
      duration: 0.5,
      stagger: 0.12,
      ease: "power2.out",
      scrollTrigger: { trigger: root, start: "top 78%" },
    })

    gsap.from(q("[data-footer-cta]"), {
      scale: 0.85,
      autoAlpha: 0,
      duration: 0.7,
      ease: "back.out(1.8)",
      scrollTrigger: { trigger: root, start: "top 70%" },
    })

    // Decorative paws drift upward through the footer.
    gsap.to(q("[data-footer-paw]"), {
      yPercent: -40,
      ease: "none",
      scrollTrigger: {
        trigger: root,
        start: "top bottom",
        end: "bottom top",
        scrub: true,
      },
    })
  })

  return (
    <footer ref={scope} className="relative overflow-hidden px-6 pt-16 pb-12">
      <div className="pointer-events-none absolute inset-0" aria-hidden="true">
        {driftingPaws.map((paw, index) => (
          <span
            key={index}
            data-footer-paw
            className={`absolute ${paw.className}`}
          >
            <Paw
              className="animate-float size-full opacity-60"
              style={{ animationDelay: paw.delay }}
            />
          </span>
        ))}
      </div>

      <div className="relative mx-auto max-w-3xl text-center">
        <div className="mx-auto mb-6 w-fit">
          <CatFace className="animate-wiggle size-24" />
        </div>
        <h2 className="text-3xl font-bold text-kitty-800 sm:text-4xl">
          {headingWords.map((word, index) => (
            <span key={index} data-footer-word className="inline-block">
              {word}
              {index < headingWords.length - 1 ? " " : ""}
            </span>
          ))}
        </h2>
        <p className="mx-auto mt-3 max-w-md text-kitty-600">
          Join the teams that traded scheduling chaos for a happy little cat.
        </p>
        <div data-footer-cta className="mt-7 flex justify-center">
          <Button size="lg">
            Adopt Lead Cat
            <Paw className="size-5" />
          </Button>
        </div>

        <div className="mt-14 flex flex-col items-center gap-3 text-sm text-kitty-400">
          <div className="flex items-center gap-2 font-bold text-kitty-600">
            <Paw className="size-5 text-coral-400" />
            Lead Cat
          </div>
          <p>
            &copy; {new Date().getFullYear()} Lead Cat. Made with whiskers and
            warmth.
          </p>
        </div>
      </div>
    </footer>
  )
}
