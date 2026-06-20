import { LeadCatLogo } from "@leadcat/brand"
import { Link } from "react-router"

import { GetStartedButton } from "~/features/landing/components/get-started-button"
import { LanguageSwitcher } from "~/shared/i18n/language-switcher"
import { useLocale, useT } from "~/shared/i18n/context"
import { localePath } from "~/shared/i18n/locale-path"
import { gsap, useSceneMotion } from "~/features/landing/lib/motion"

export function Nav() {
  const t = useT()
  const locale = useLocale()

  const scope = useSceneMotion<HTMLElement>((header) => {
    gsap.to(header, {
      paddingTop: "0.6rem",
      paddingBottom: "0.6rem",
      backgroundColor: "var(--card)",
      backdropFilter: "blur(16px)",
      boxShadow: "var(--shadow-card)",
      borderColor: "var(--border)",
      duration: 0.3,
      ease: "power2.out",
      scrollTrigger: {
        start: 24,
        end: "max",
        toggleActions: "play none none reverse",
      },
    })
  })

  return (
    <header
      ref={scope}
      className="sticky top-0 z-30 border-b border-transparent py-6"
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-6">
        <Link to={localePath(locale)} className="inline-flex">
          <LeadCatLogo />
        </Link>
        <nav className="hidden items-center gap-8 text-sm font-semibold text-kitty-600 md:flex">
          <a
            className="transition-colors hover:text-coral-500"
            href="#features"
          >
            {t("nav.features")}
          </a>
          <a className="transition-colors hover:text-coral-500" href="#how">
            {t("nav.howItWorks")}
          </a>
          <a
            className="transition-colors hover:text-coral-500"
            href="#showcase"
          >
            {t("nav.miniApp")}
          </a>
        </nav>
        <div className="flex items-center gap-3">
          <LanguageSwitcher />
          <GetStartedButton size="sm">{t("nav.getStarted")}</GetStartedButton>
        </div>
      </div>
    </header>
  )
}
