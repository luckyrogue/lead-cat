import { Link } from "@tanstack/react-router"
import { WEEKDAYS } from "@/shared/tma/constants"
import { useTmaApp } from "@/shared/tma/context"
import type { Lang, Scenario } from "@/shared/tma/types"
import { hexToRgba } from "@/shared/tma/palette"
import { CatCard, CatIcon, CatToggle } from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id"

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

export function AutoScreen({
  scenarios,
  onToggle,
}: {
  scenarios: Scenario[]
  onToggle: (id: string) => void
}) {
  const p = useTmaApp()
  const t = p.t
  const workspaceId = useWorkspaceId()

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <h2
        style={{
          margin: "2px 4px 4px",
          fontFamily: "var(--font-display)",
          fontWeight: 800,
          fontSize: 26,
          color: p.text,
        }}
      >
        {t("nav_auto")} ⚡
      </h2>
      <p style={{ margin: "0 4px 18px", color: p.muted, fontSize: 14 }}>
        Кот сам напомнит, пингнёт и пришлёт отчёт
      </p>

      <div
        style={{
          position: "relative",
          overflow: "hidden",
          borderRadius: 20,
          padding: "16px 18px",
          marginBottom: 18,
          background: `linear-gradient(135deg, oklch(${p.dark ? 0.34 : 0.95} 0.07 25), oklch(${p.dark ? 0.32 : 0.96} 0.05 45))`,
          border: `1px solid ${p.border}`,
        }}
      >
        <Paw
          size={90}
          color={hexToRgba(p.accent, 0.12)}
          style={{
            position: "absolute",
            right: -16,
            top: -16,
            transform: "rotate(20deg)",
          }}
        />
        <div
          style={{
            position: "relative",
            display: "flex",
            alignItems: "center",
            gap: 12,
          }}
        >
          <span style={{ fontSize: 30 }}>🤖</span>
          <div>
            <div
              style={{
                fontFamily: "var(--font-display)",
                fontWeight: 800,
                fontSize: 15.5,
                color: p.text,
              }}
            >
              {scenarios.filter((s) => s.enabled).length} активных сценария
            </div>
            <div style={{ fontSize: 13, color: p.muted, fontWeight: 600 }}>
              работают по расписанию в чате команды
            </div>
          </div>
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {scenarios.map((s) => {
          const days =
            s.trigger.days.length === 5 &&
            s.trigger.days.every((d, i) => d === i + 1)
              ? "Пн–Пт"
              : s.trigger.days.map((d) => WEEKDAYS[d - 1]).join(", ")
          return (
            <CatCard
              key={s.id}
              pad={0}
              style={{ overflow: "hidden", opacity: s.enabled ? 1 : 0.62 }}
            >
              <div style={{ padding: "14px 15px" }}>
                <div
                  style={{
                    display: "flex",
                    alignItems: "flex-start",
                    gap: 10,
                    marginBottom: 12,
                  }}
                >
                  <div style={{ flex: 1 }}>
                    <div
                      style={{
                        fontFamily: "var(--font-display)",
                        fontWeight: 800,
                        fontSize: 16,
                        color: p.text,
                        marginBottom: 4,
                      }}
                    >
                      {s.name}
                    </div>
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 8,
                        color: p.muted,
                        fontSize: 13,
                        fontWeight: 600,
                      }}
                    >
                      <span
                        style={{
                          display: "inline-flex",
                          alignItems: "center",
                          gap: 4,
                        }}
                      >
                        <CatIcon
                          name="clock"
                          size={14}
                          color={p.muted}
                          sw={2}
                        />
                        {String(s.trigger.hour).padStart(2, "0")}:
                        {String(s.trigger.minute).padStart(2, "0")}
                      </span>
                      <span
                        style={{
                          display: "inline-flex",
                          alignItems: "center",
                          gap: 4,
                        }}
                      >
                        <CatIcon
                          name="repeat"
                          size={13}
                          color={p.muted}
                          sw={2}
                        />
                        {days}
                      </span>
                    </div>
                  </div>
                  <CatToggle on={s.enabled} onChange={() => onToggle(s.id)} />
                </div>
                <div
                  style={{
                    fontSize: 13,
                    color: p.muted,
                    lineHeight: 1.4,
                    marginBottom: 12,
                  }}
                >
                  {s.note}
                </div>
                <div style={{ display: "flex", gap: 7, flexWrap: "wrap" }}>
                  {s.actions.map((act) => {
                    const meta = ACTION_META[act]
                    if (!meta) return null
                    return (
                      <span
                        key={act}
                        style={{
                          display: "inline-flex",
                          alignItems: "center",
                          gap: 5,
                          background: p.cardAlt,
                          border: `1px solid ${p.border}`,
                          borderRadius: 999,
                          padding: "4px 10px",
                          fontSize: 12.5,
                          fontWeight: 700,
                          color: p.text,
                          fontFamily: "var(--font-display)",
                        }}
                      >
                        <span style={{ fontSize: 13 }}>{meta.emoji}</span>
                        {meta.label[p.lang]}
                      </span>
                    )
                  })}
                </div>
              </div>
            </CatCard>
          )
        })}
      </div>

      {workspaceId ? (
        <Link
          to="/scenarios"
          search={{ workspaceId }}
          style={{
            marginTop: 14,
            width: "100%",
            padding: "15px",
            borderRadius: 16,
            border: `1.5px dashed ${p.borderStrong}`,
            background: "transparent",
            color: p.accent,
            fontWeight: 800,
            fontFamily: "var(--font-display)",
            fontSize: 15,
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            gap: 8,
            textDecoration: "none",
          }}
        >
          <CatIcon name="plus" size={20} color={p.accent} sw={2.4} />{" "}
          {t("legacyScenarios")}
        </Link>
      ) : (
        <button
          type="button"
          onClick={() => p.showToast("Конструктор сценариев скоро 🐾")}
          style={{
            marginTop: 14,
            width: "100%",
            padding: "15px",
            borderRadius: 16,
            border: `1.5px dashed ${p.borderStrong}`,
            background: "transparent",
            color: p.accent,
            fontWeight: 800,
            fontFamily: "var(--font-display)",
            fontSize: 15,
            cursor: "pointer",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            gap: 8,
          }}
        >
          <CatIcon name="plus" size={20} color={p.accent} sw={2.4} /> Новый
          сценарий
        </button>
      )}
    </div>
  )
}
