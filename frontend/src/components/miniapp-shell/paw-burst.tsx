export function PawBurst({ show }: { show: boolean }) {
  if (!show) return null
  const bits = Array.from({ length: 14 })
  return (
    <div className="pointer-events-none absolute inset-0 z-[95] overflow-hidden">
      {bits.map((_, i) => {
        const ang = (i / bits.length) * Math.PI * 2
        const dist = 90 + (i % 4) * 36
        const hue = [25, 150, 300, 180, 95][i % 5]
        const style = {
          position: "absolute" as const,
          left: "50%",
          top: "42%",
          fontSize: 18 + (i % 3) * 6,
          color: `oklch(0.65 0.15 ${hue})`,
          animation: "lc-burst .9s cubic-bezier(.2,.7,.3,1) forwards",
          animationDelay: `${(i % 5) * 18}ms`,
          ["--tx" as string]: `${Math.cos(ang) * dist}px`,
          ["--ty" as string]: `${Math.sin(ang) * dist}px`,
          ["--rot" as string]: `${(i % 2 ? 1 : -1) * (120 + i * 12)}deg`,
        }
        return (
          <span key={i} style={style}>
            🐾
          </span>
        )
      })}
    </div>
  )
}
