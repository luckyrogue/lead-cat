import { Features } from "~/features/landing/components/features"
import { Footer } from "~/features/landing/components/footer"
import { Hero } from "~/features/landing/components/hero"
import { HowItWorks } from "~/features/landing/components/how-it-works"
import { Nav } from "~/features/landing/components/nav"
import { Showcase } from "~/features/landing/components/showcase"
import { TrustMarquee } from "~/features/landing/components/trust-marquee"

export function LandingPage() {
  return (
    <main className="relative min-h-svh">
      <Nav />
      <Hero />
      <TrustMarquee />
      <Features />
      <HowItWorks />
      <Showcase />
      <Footer />
    </main>
  )
}
