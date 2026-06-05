import { useTmaApp } from "@/shared/tma/context"

export function CatToggle({
  on,
  onChange,
}: {
  on: boolean
  onChange: (v: boolean) => void
}) {
  const p = useTmaApp()
  return (
    <button
      type="button"
      onClick={() => onChange(!on)}
      style={{
        width: 50,
        height: 30,
        borderRadius: 999,
        border: "none",
        cursor: "pointer",
        background: on
          ? p.accent
          : p.dark
            ? "rgba(255,255,255,0.16)"
            : "#E4D7C8",
        position: "relative",
        transition: "background .22s ease",
        flexShrink: 0,
        padding: 0,
      }}
    >
      <span
        style={{
          position: "absolute",
          top: 3,
          left: on ? 23 : 3,
          width: 24,
          height: 24,
          borderRadius: "50%",
          background: "#fff",
          transition: "left .22s cubic-bezier(.34,1.56,.64,1)",
          boxShadow: "0 1px 4px rgba(0,0,0,0.25)",
        }}
      />
    </button>
  )
}
