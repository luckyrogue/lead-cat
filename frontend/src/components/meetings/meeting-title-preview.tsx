import { useTmaApp } from "@/shared/tma/context"

export function MeetingTitlePreview({
  label,
  title,
  marginBottom = 16,
}: {
  label: string
  title: string
  marginBottom?: number
}) {
  const p = useTmaApp()
  return (
    <div
      style={{
        background: p.cardAlt,
        borderRadius: 16,
        padding: "13px 15px",
        marginBottom,
        border: `1px dashed ${p.borderStrong}`,
      }}
    >
      <div
        style={{
          fontSize: 11,
          color: p.faint,
          fontWeight: 700,
          marginBottom: 5,
          letterSpacing: 0.3,
          textTransform: "uppercase",
        }}
      >
        {label}
      </div>
      <div
        style={{
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 17,
          color: p.text,
          lineHeight: 1.25,
          wordBreak: "break-word",
        }}
      >
        {title}
      </div>
    </div>
  )
}
