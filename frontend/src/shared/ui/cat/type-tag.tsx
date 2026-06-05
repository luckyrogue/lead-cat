import { typeAccent } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"

export function TypeTag({
  typeKey,
  label,
  size = "md",
}: {
  typeKey: string
  label: string
  size?: "sm" | "md"
}) {
  const { dark } = useTmaApp()
  const a = typeAccent(typeKey, dark)
  const s =
    size === "sm"
      ? { fs: 11.5, px: 8, py: 3, gap: 4 }
      : { fs: 13, px: 10, py: 5, gap: 5 }
  return (
    <span
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: s.gap,
        background: a.soft,
        color: a.text,
        padding: `${s.py}px ${s.px}px`,
        borderRadius: 999,
        fontSize: s.fs,
        fontWeight: 700,
        fontFamily: "var(--font-display)",
        lineHeight: 1,
        whiteSpace: "nowrap",
      }}
    >
      <span style={{ fontSize: s.fs }}>{a.emoji}</span>
      {label}
    </span>
  )
}
