import { useNavigate } from "@tanstack/react-router"
import type { Employee } from "@/entities/employee/types"
import type { FreeSlot } from "@/entities/meeting/types"
import { useTmaApp } from "@/shared/tma/context"
import { EmptyState } from "@/components/meetings/meeting-ui"
import { CatCard, CatIcon } from "@/shared/ui/cat/primitives"

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
  const t = useTmaApp().t
  const navigate = useNavigate()

  if (isError) {
    return (
      <div className="mt-3 text-center text-sm font-semibold text-tma-muted">
        Не удалось найти слоты. Попробуй ещё раз.
      </div>
    )
  }

  if (slots.length === 0) {
    return (
      <div className="mt-[22px]">
        <div className="overflow-hidden rounded-[20px] border border-tma-border bg-tma-card">
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
    <div className="mt-[22px]">
      <div className="font-display mx-1 mb-3 flex items-center gap-[7px] text-[15px] font-extrabold text-tma-ok">
        ✅ {t("freeFor")} {people.length} {t("people")}
      </div>
      <div className="flex flex-col gap-2.5">
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
            className="flex items-center gap-[13px] p-3.5"
          >
            <div className="flex size-[46px] shrink-0 flex-col items-center justify-center rounded-[13px] bg-tma-ok-soft">
              <span className="text-base">📅</span>
            </div>
            <div className="flex-1">
              <div className="font-display text-[15.5px] font-extrabold text-tma-text">
                {s.start} – {s.end}
              </div>
              <div className="text-[13px] font-semibold text-tma-muted">
                {s.day} · {s.mins} {t("min")}
              </div>
            </div>
            <div className="font-display flex items-center gap-1 text-[13px] font-bold text-tma-accent">
              {t("createHere")}
              <CatIcon name="chevR" size={15} sw={2.4} />
            </div>
          </CatCard>
        ))}
      </div>
    </div>
  )
}
