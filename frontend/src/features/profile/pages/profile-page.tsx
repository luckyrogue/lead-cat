import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { toastSuccess } from "@/shared/lib/toast"
import { I18N } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { canAccessTmaAdmin } from "@/shared/auth/module-policies"
import { useTmaAuth } from "@/features/auth/auth-context"
import { CatIcon, CatToggle } from "@/shared/ui/cat/primitives"
import { ProfileHeader } from "../components/profile-header"
import { SettingsGroup } from "../components/settings-group"
import { SettingsRow } from "../components/settings-row"

export function ProfilePage() {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const navigate = useNavigate()
  const t = p.t
  const [reminders, setReminders] = useState(["15m"])
  const [remOn, setRemOn] = useState(true)
  const intervals = [
    { v: "10m", l: `10 ${t("min")}` },
    { v: "15m", l: `15 ${t("min")}` },
    { v: "30m", l: `30 ${t("min")}` },
    { v: "1h", l: `1 ${t("hour")}` },
    { v: "2h", l: `2 ${t("hour")}` },
    { v: "1d", l: "1 день" },
  ]

  return (
    <div style={{ padding: "16px 16px 28px" }}>
      <ProfileHeader />

      <SettingsGroup title={t("reminders")}>
        <SettingsRow
          icon="bell"
          hue={45}
          label={t("reminders")}
          right={<CatToggle on={remOn} onChange={setRemOn} />}
          last
        />
      </SettingsGroup>
      {remOn && (
        <div
          style={{
            margin: "-12px 4px 20px",
            display: "flex",
            flexWrap: "wrap",
            gap: 8,
          }}
        >
          {intervals.map((it) => {
            const on = reminders.includes(it.v)
            return (
              <button
                key={it.v}
                type="button"
                onClick={() =>
                  setReminders(
                    on
                      ? reminders.filter((x) => x !== it.v)
                      : [...reminders, it.v]
                  )
                }
                style={{
                  padding: "8px 13px",
                  borderRadius: 11,
                  border: `1.5px solid ${on ? p.accent : p.border}`,
                  background: on ? p.accentSoft : p.card,
                  color: on ? p.accent : p.muted,
                  fontWeight: 700,
                  fontSize: 13.5,
                  fontFamily: "var(--font-display)",
                  cursor: "pointer",
                }}
              >
                {on ? "✓ " : ""}
                {it.l}
              </button>
            )
          })}
        </div>
      )}

      <SettingsGroup title="Предпочтения">
        <SettingsRow
          icon="globe"
          hue={180}
          label={t("language")}
          onClick={p.openLangPicker}
          right={
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                color: p.muted,
                fontWeight: 700,
                fontSize: 14,
              }}
            >
              {I18N[p.lang]._flag} {I18N[p.lang]._label}
              <CatIcon name="chevR" size={16} color={p.faint} sw={2.2} />
            </span>
          }
        />
        <SettingsRow
          icon="clock"
          hue={300}
          label={t("timezone")}
          right={
            <span
              style={{
                display: "flex",
                alignItems: "center",
                gap: 6,
                color: p.muted,
                fontWeight: 700,
                fontSize: 14,
              }}
            >
              Алматы UTC+5
              <CatIcon name="chevR" size={16} color={p.faint} sw={2.2} />
            </span>
          }
          last
        />
      </SettingsGroup>

      {canAccessTmaAdmin(user) && (
        <SettingsGroup title={t("admin")}>
          <SettingsRow
            icon="shield"
            hue={25}
            label={t("admin")}
            onClick={() => void navigate({ to: "/profile/admin" })}
            right={<CatIcon name="chevR" size={18} color={p.faint} sw={2.2} />}
            last
          />
        </SettingsGroup>
      )}

      <SettingsGroup>
        <SettingsRow
          icon="sparkle"
          hue={95}
          label={t("help")}
          onClick={() => toastSuccess("Мяу! Чем помочь? 🐾")}
          right={<CatIcon name="chevR" size={18} color={p.faint} sw={2.2} />}
          last
        />
      </SettingsGroup>

      <div
        style={{
          textAlign: "center",
          marginTop: 8,
          color: p.faint,
          fontSize: 12,
          fontWeight: 600,
        }}
      >
        Lead Cat · v2.0 · {I18N[p.lang].appSub} 🐾
      </div>
    </div>
  )
}
