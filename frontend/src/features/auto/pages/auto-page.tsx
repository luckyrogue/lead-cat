import { useState } from "react"
import { cn } from "@/shared/lib/cn"
import { toastSuccess } from "@/shared/lib/toast"
import { WEEKDAYS } from "@/entities/meeting/constants"
import { INITIAL_SCENARIOS } from "@/features/auto/fixtures"
import { useTmaApp } from "@/shared/tma/context"
import type { Lang } from "@/shared/tma/types"
import { CatCard, CatIcon, CatToggle } from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"

const ACTION_META: Record<
  string,
  { emoji: string; label: Record<Lang, string> }
> = {
  message: {
    emoji: "💬",
    label: { ru: "Сообщение", kk: "Хабарлама", en: "Message" },
  },
  cat_photo: {
    emoji: "🐱",
    label: { ru: "Котофото", kk: "Мысық фото", en: "Cat photo" },
  },
  commits_report: {
    emoji: "📊",
    label: { ru: "Отчёт коммитов", kk: "Коммит есебі", en: "Commits report" },
  },
}

export function AutoPage() {
  const { t, lang } = useTmaApp()
  const [scenarios, setScenarios] = useState(INITIAL_SCENARIOS)
  const toggleScenario = (id: string) =>
    setScenarios((arr) =>
      arr.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s))
    )

  return (
    <div className="px-4 pb-7">
      <h2 className="tma-heading mx-1 mt-0.5 mb-1 text-[26px]">
        {t("nav_auto")} ⚡
      </h2>
      <p className="mx-1 mb-[18px] text-sm text-tma-muted">
        Кот сам напомнит, пингнёт и пришлёт отчёт
      </p>

      <div className="relative mb-[18px] overflow-hidden rounded-[20px] border border-tma-border bg-tma-auto-banner px-[18px] py-4">
        <Paw
          size={90}
          tone="auto"
          className="absolute -top-4 -right-4 rotate-[20deg]"
        />
        <div className="relative flex items-center gap-3">
          <span className="text-[30px]">🤖</span>
          <div>
            <div className="font-display text-[15.5px] font-extrabold text-tma-text">
              {scenarios.filter((s) => s.enabled).length} активных сценария
            </div>
            <div className="text-[13px] font-semibold text-tma-muted">
              работают по расписанию в чате команды
            </div>
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        {scenarios.map((s) => {
          const days =
            s.trigger.days.length === 5 &&
            s.trigger.days.every((d, i) => d === i + 1)
              ? "Пн–Пт"
              : s.trigger.days.map((d) => WEEKDAYS[d - 1]).join(", ")
          return (
            <CatCard
              key={s.id}
              className={cn("overflow-hidden p-0", !s.enabled && "opacity-[0.62]")}
            >
              <div className="px-[15px] py-3.5">
                <div className="mb-3 flex items-start gap-2.5">
                  <div className="flex-1">
                    <div className="font-display mb-1 text-base font-extrabold text-tma-text">
                      {s.name}
                    </div>
                    <div className="flex items-center gap-2 text-[13px] font-semibold text-tma-muted">
                      <span className="inline-flex items-center gap-1">
                        <CatIcon
                          name="clock"
                          size={14}
                          className="text-tma-muted"
                          sw={2}
                        />
                        {String(s.trigger.hour).padStart(2, "0")}:
                        {String(s.trigger.minute).padStart(2, "0")}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <CatIcon
                          name="repeat"
                          size={13}
                          className="text-tma-muted"
                          sw={2}
                        />
                        {days}
                      </span>
                    </div>
                  </div>
                  <CatToggle
                    on={s.enabled}
                    onChange={() => toggleScenario(s.id)}
                  />
                </div>
                <div className="mb-3 text-[13px] leading-snug text-tma-muted">
                  {s.note}
                </div>
                <div className="flex flex-wrap gap-[7px]">
                  {s.actions.map((act) => {
                    const meta = ACTION_META[act]
                    if (!meta) return null
                    return (
                      <span
                        key={act}
                        className="font-display inline-flex items-center gap-[5px] rounded-full border border-tma-border bg-tma-card-alt px-2.5 py-1 text-[12.5px] font-bold text-tma-text"
                      >
                        <span className="text-[13px]">{meta.emoji}</span>
                        {meta.label[lang]}
                      </span>
                    )
                  })}
                </div>
              </div>
            </CatCard>
          )
        })}
      </div>

      <button
        type="button"
        onClick={() => toastSuccess("Конструктор сценариев скоро 🐾")}
        className="font-display mt-3.5 flex w-full cursor-pointer items-center justify-center gap-2 rounded-2xl border-[1.5px] border-dashed border-tma-border-strong bg-transparent px-[15px] py-[15px] text-[15px] font-extrabold text-tma-accent"
      >
        <CatIcon name="plus" size={20} className="text-tma-accent" sw={2.4} />{" "}
        Новый сценарий
      </button>
    </div>
  )
}
