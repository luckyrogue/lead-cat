import { PawIcon, motion } from "@leadcat/ui";

function Sparkle({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden="true">
      <path d="M12 0c.6 5.4 2 6.8 7.4 7.4C14 8 12.6 9.4 12 14.8 11.4 9.4 10 8 4.6 7.4 10 6.8 11.4 5.4 12 0z" />
    </svg>
  );
}

const motifs = [
  { Comp: PawIcon, top: "12%", left: "6%", size: 46, color: "text-peach-300", dur: 7, delay: 0 },
  { Comp: PawIcon, top: "70%", left: "10%", size: 32, color: "text-coral-200", dur: 6, delay: 1.2 },
  { Comp: Sparkle, top: "22%", left: "88%", size: 28, color: "text-sunny-300", dur: 5, delay: 0.5 },
  { Comp: PawIcon, top: "82%", left: "84%", size: 40, color: "text-peach-300", dur: 8, delay: 0.8 },
  { Comp: Sparkle, top: "55%", left: "94%", size: 22, color: "text-coral-200", dur: 6.5, delay: 1.6 },
  { Comp: Sparkle, top: "40%", left: "3%", size: 24, color: "text-sunny-300", dur: 5.5, delay: 0.3 },
];

export function FloatingMotifs() {
  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      {motifs.map(({ Comp, top, left, size, color, dur, delay }, i) => (
        <motion.div
          key={i}
          className={color}
          style={{ position: "absolute", top, left, width: size, height: size }}
          animate={{ y: [0, -18, 0], rotate: [0, 12, 0] }}
          transition={{ duration: dur, delay, repeat: Infinity, ease: "easeInOut" }}
        >
          <Comp className="h-full w-full" />
        </motion.div>
      ))}
    </div>
  );
}
