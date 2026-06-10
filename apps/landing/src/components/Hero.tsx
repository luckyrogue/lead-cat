import { lazy, Suspense } from "react";
import { Badge, Button, PawIcon, motion } from "@leadcat/ui";
import { FloatingMotifs } from "./FloatingMotifs";

const CatScene = lazy(() =>
  import("@leadcat/ui/3d").then((m) => ({ default: m.CatScene })),
);

function SceneFallback() {
  return (
    <div className="grid h-full w-full place-items-center">
      <div className="flex flex-col items-center gap-3 text-coral-400">
        <PawIcon className="h-14 w-14 animate-bounce-soft" />
        <span className="text-sm font-semibold">Waking the kitty…</span>
      </div>
    </div>
  );
}

export function Hero() {
  return (
    <section className="relative overflow-hidden px-6 pb-12 pt-8 md:pt-16">
      <FloatingMotifs />
      <div className="relative mx-auto grid max-w-6xl items-center gap-8 md:grid-cols-2 md:gap-4">
        <motion.div
          initial={{ opacity: 0, y: 28 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: [0.34, 1.56, 0.64, 1] }}
          className="text-center md:text-left"
        >
          <motion.div
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ delay: 0.15, type: "spring", stiffness: 260, damping: 16 }}
            className="mb-5 inline-block"
          >
            <Badge tone="sunny">🐾 Scheduling, but cuddly</Badge>
          </motion.div>
          <h1 className="text-4xl font-bold leading-tight text-kitty-800 sm:text-5xl lg:text-6xl">
            Meetings your team will{" "}
            <span className="relative whitespace-nowrap text-coral-500">
              actually love
              <motion.span
                className="absolute -bottom-2 left-0 h-3 w-full rounded-full bg-sunny-300/70"
                initial={{ scaleX: 0 }}
                animate={{ scaleX: 1 }}
                transition={{ delay: 0.7, duration: 0.5, ease: "easeOut" }}
                style={{ originX: 0 }}
              />
            </span>{" "}
            🐾
          </h1>
          <p className="mx-auto mt-6 max-w-md text-lg text-kitty-600 md:mx-0">
            Lead Cat sniffs out the purrfect time across everyone's calendars — no
            more back-and-forth, no more double-booking. Just happy little meetings.
          </p>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-4 md:justify-start">
            <Button size="lg">Get started — it's free</Button>
            <Button variant="secondary" size="lg">
              See how it works
            </Button>
          </div>
          <p className="mt-5 text-sm font-semibold text-kitty-400">
            Works with Google Calendar & Telegram. No credit card, no claws out.
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, scale: 0.85 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.2, duration: 0.7, ease: [0.34, 1.56, 0.64, 1] }}
          className="relative mx-auto aspect-square w-full max-w-md"
        >
          <div className="absolute inset-4 rounded-blob bg-gradient-to-br from-peach-200 via-cream-300 to-sunny-200 blur-2xl" />
          <div className="relative h-full w-full animate-purr">
            <Suspense fallback={<SceneFallback />}>
              <CatScene followPointer />
            </Suspense>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
