import { Paw } from "@leadcat/ui"

const motifs = [
  { className: "left-[6%] top-[18%] size-10 text-coral-200", delay: "0s" },
  { className: "right-[10%] top-[12%] size-8 text-sunny-300", delay: "0.6s" },
  { className: "left-[14%] bottom-[14%] size-7 text-peach-300", delay: "1.1s" },
  {
    className: "right-[16%] bottom-[22%] size-9 text-coral-200",
    delay: "1.6s",
  },
]

export function FloatingMotifs() {
  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden">
      {motifs.map((motif, index) => (
        <Paw
          key={index}
          className={`absolute animate-float opacity-70 ${motif.className}`}
          style={{ animationDelay: motif.delay }}
        />
      ))}
    </div>
  )
}
