import { useNavigate } from "@tanstack/react-router"
import type { Employee } from "@/entities/employee/types"
import { useTmaApp } from "@/shared/tma/context"
import { EmptyState } from "@/features/meetings/components/meeting-ui"
import { CatCard, CatIcon } from "@/shared/ui/cat/primitives"

type FreeSlot = {
  iso: string
  start: string
  end: string
  day: string
  mins: number
}

export function CheckerSlotsResults({
  people,
  slots,
  isError,
}: {
  people: Employee[]
  dur: number
  slots: FreeSlot[]
  isError: boolean
}) {
  const p = useTmaApp()
  const t = p.t
  const navigate = useNavigate()

  if (isError) {
    return (
      <div
        style={{
          marginTop: 12,
          color: p.muted,
          fontSize: 14,
          fontWeight: 600,
          textAlign: "center",
        }}
      >
        Не удалось найти слоты. Попробуй ещё раз.
      </div>
    )
  }

  if (slots.length === 0) {
    return (
      <div style={{ marginTop: 22 }}>
        <div
          style={{
            background: p.card,
            borderRadius: 20,
            border: `1px solid ${p.border}`,
            overflow: "hidden",
          }}
        >
          <EmptyState
            emoji="🙀"
            title={t("noSlots")}
            sub="Попробуй расширить диапазон или уменьшить длительность"
          />
        </div>
      </div>
    )
  }

  return (
    <div style={{ marginTop: 22 }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 7,
          color: p.ok,
          fontWeight: 800,
          fontFamily: "var(--font-display)",
          fontSize: 15,
          margin: "0 4px 12px",
        }}
      >
        ✅ {t("freeFor")} {people.length} {t("people")}
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        {slots.map((s, i) => (
          <CatCard
            key={i}
            interactive
            onClick={() => {
              sessionStorage.setItem(
                "tma-create-initial",
                JSON.stringify({
                  date: s.iso,
                  start: s.start,
                  dur: s.mins >= 120 ? 60 : s.mins,
                  participants: people,
                })
              )
              void navigate({ to: "/meetings/create" })
            }}
            pad={14}
            style={{ display: "flex", alignItems: "center", gap: 13 }}
          >
            <div
              style={{
                width: 46,
                height: 46,
                borderRadius: 13,
                background: p.okSoft,
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
              }}
            >
              <span style={{ fontSize: 16 }}>📅</span>
            </div>
            <div style={{ flex: 1 }}>
              <div
                style={{
                  fontFamily: "var(--font-display)",
                  fontWeight: 800,
                  fontSize: 15.5,
                  color: p.text,
                }}
              >
                {s.start} – {s.end}
              </div>
              <div
                style={{
                  fontSize: 13,
                  color: p.muted,
                  fontWeight: 600,
                }}
              >
                {s.day} · {s.mins} {t("min")}
              </div>
            </div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 4,
                color: p.accent,
                fontWeight: 700,
                fontSize: 13,
                fontFamily: "var(--font-display)",
              }}
            >
              {t("createHere")}
              <CatIcon name="chevR" size={15} color={p.accent} sw={2.4} />
            </div>
          </CatCard>
        ))}
      </div>
    </div>
  )
}
