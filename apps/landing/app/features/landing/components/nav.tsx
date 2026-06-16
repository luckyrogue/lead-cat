import { Button, Paw } from "@leadcat/ui"

import { gsap, useSceneMotion } from "~/features/landing/lib/motion"

export function Nav() {
  // Sticky always (layout); the condense-on-scroll is motion-gated.
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
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6">
        <a
          href="/"
          className="flex items-center gap-2 font-bold text-kitty-800"
        >
          <span className="grid size-9 place-items-center rounded-2xl bg-coral-400 text-white">
            <Paw className="size-5" />
          </span>
          Lead Cat
        </a>
        <nav className="hidden items-center gap-8 text-sm font-semibold text-kitty-600 md:flex">
          <a
            className="transition-colors hover:text-coral-500"
            href="#features"
          >
            Features
          </a>
          <a className="transition-colors hover:text-coral-500" href="#how">
            How it works
          </a>
          <a
            className="transition-colors hover:text-coral-500"
            href="#showcase"
          >
            Mini App
          </a>
        </nav>
        <Button size="sm">Get started</Button>
      </div>
    </header>
  )
}
