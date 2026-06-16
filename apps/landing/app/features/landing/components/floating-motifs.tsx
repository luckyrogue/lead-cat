import { Paw } from "@leadcat/ui"

// `parallax` scales how far each paw drifts with the pointer; the wrapper span
// carries the GSAP transform while the inner Paw keeps the CSS float loop, so
// the two never fight over `transform`.
const motifs = [
  {
    className: "left-[6%] top-[18%] size-10 text-coral-200",
    delay: "0s",
    parallax: 0.6,
  },
  {
    className: "right-[10%] top-[12%] size-8 text-sunny-300",
    delay: "0.6s",
    parallax: 1.3,
  },
  {
    className: "left-[14%] bottom-[14%] size-7 text-peach-300",
    delay: "1.1s",
    parallax: 0.9,
  },
  {
    className: "right-[16%] bottom-[22%] size-9 text-coral-200",
    delay: "1.6s",
    parallax: 1.6,
  },
]

export function FloatingMotifs() {
  return (
    <div
      data-hero-motifs
      className="pointer-events-none absolute inset-0 overflow-hidden"
      aria-hidden="true"
    >
      {motifs.map((motif, index) => (
        <span
          key={index}
          data-parallax={motif.parallax}
          className={`absolute ${motif.className}`}
        >
          <Paw
            className="animate-float size-full opacity-70"
            style={{ animationDelay: motif.delay }}
          />
        </span>
      ))}
    </div>
  )
}
