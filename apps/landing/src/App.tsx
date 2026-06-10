import { PawIcon, motion } from "@leadcat/ui";
import { Hero } from "./components/Hero";
import { Features } from "./components/Features";
import { HowItWorks } from "./components/HowItWorks";
import { Footer } from "./components/Footer";

function Nav() {
  return (
    <motion.nav
      initial={{ opacity: 0, y: -16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
      className="mx-auto flex max-w-6xl items-center justify-between px-6 py-6"
    >
      <div className="flex items-center gap-2 text-xl font-bold text-kitty-800">
        <span className="grid h-9 w-9 place-items-center rounded-full bg-coral-500 text-white shadow-pop">
          <PawIcon className="h-5 w-5" />
        </span>
        Lead Cat
      </div>
      <a
        href="#"
        className="rounded-full px-5 py-2 text-sm font-bold text-coral-600 transition-colors hover:bg-coral-50"
      >
        Sign in
      </a>
    </motion.nav>
  );
}

export function App() {
  return (
    <div className="min-h-screen bg-gradient-to-b from-cream-100 via-peach-50 to-cream-200 font-sans text-kitty-800">
      <Nav />
      <Hero />
      <Features />
      <HowItWorks />
      <Footer />
    </div>
  );
}
