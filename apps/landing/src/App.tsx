import { lazy, Suspense } from "react";
import { Button } from "@leadcat/ui";
import { motion } from "motion/react";

const R3FScene = lazy(() =>
  import("@leadcat/ui/3d").then((m) => ({ default: m.R3FScene })),
);

export function App() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-8 bg-surface-soft p-8">
      <motion.h1
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6, ease: "easeOut" }}
        className="text-center text-4xl font-extrabold text-brand-700"
      >
        Meetings, made friendly.
      </motion.h1>
      <Button>Try Lead Cat</Button>
      <div className="h-72 w-full max-w-lg overflow-hidden rounded-3xl shadow-glow">
        <Suspense fallback={<div className="grid h-full place-items-center text-brand-400">Loading scene…</div>}>
          <R3FScene />
        </Suspense>
      </div>
    </main>
  );
}
