import { CatFace, Paw } from "@leadcat/ui"
import { LeadCatLogo, SITE_NAME } from "@leadcat/brand"

import { GetStartedButton } from "~/features/landing/components/get-started-button"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"
import { useT } from "~/shared/i18n/context"
import { useLandingDict } from "~/shared/i18n/use-landing-dict"

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
  const t = useT()
  const dict = useLandingDict()

  const scope = useSceneMotion<HTMLElement>((root) => {
    const q = gsap.utils.selector(root)

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
          {dict.footer.headingWords.map((word, index) => (
            <span key={index} data-footer-word className="inline-block">
              {word}
              {index < dict.footer.headingWords.length - 1 ? "\u00a0" : ""}
            </span>
          ))}
        </h2>
        <p className="mx-auto mt-3 max-w-md text-kitty-600">{t("footer.copy")}</p>
        <div data-footer-cta className="mt-7 flex justify-center">
          <GetStartedButton size="lg">
            {t("footer.cta")}
            <Paw className="size-5" />
          </GetStartedButton>
        </div>

        <div className="mt-14 flex flex-col items-center gap-3 text-sm text-kitty-400">
          <LeadCatLogo className="text-kitty-600" />
          <p>
            &copy; {new Date().getFullYear()} {SITE_NAME}. {t("footer.copyright")}
          </p>
        </div>
      </div>
    </footer>
  )
}
