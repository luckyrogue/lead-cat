import { useState } from "react"
import { useNavigate } from "@tanstack/react-router"
import { cn } from "@/shared/lib/cn"
import { toastSuccess } from "@/shared/lib/toast"
import { I18N } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { canAccessTmaAdmin } from "@/shared/auth/module-policies"
import { useTmaAuth } from "@/shared/auth/auth-context"
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
    <div className="px-4 pb-7">
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
        <div className="mx-1 -mt-3 mb-5 flex flex-wrap gap-2">
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
                className={cn(
                  "font-display cursor-pointer rounded-[11px] border-[1.5px] px-[13px] py-2 text-[13.5px] font-bold",
                  on
                    ? "border-tma-accent bg-tma-accent-soft text-tma-accent"
                    : "border-tma-border bg-tma-card text-tma-muted"
                )}
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
            <span className="text-tma-muted flex items-center gap-1.5 text-sm font-bold">
              {I18N[p.lang]._flag} {I18N[p.lang]._label}
              <span className="text-tma-faint">
                <CatIcon name="chevR" size={16} sw={2.2} />
              </span>
            </span>
          }
        />
        <SettingsRow
          icon="clock"
          hue={300}
          label={t("timezone")}
          right={
            <span className="text-tma-muted flex items-center gap-1.5 text-sm font-bold">
              Алматы UTC+5
              <span className="text-tma-faint">
                <CatIcon name="chevR" size={16} sw={2.2} />
              </span>
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
            right={
              <span className="text-tma-faint">
                <CatIcon name="chevR" size={18} sw={2.2} />
              </span>
            }
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
          right={
            <span className="text-tma-faint">
              <CatIcon name="chevR" size={18} sw={2.2} />
            </span>
          }
          last
        />
      </SettingsGroup>

      <div className="text-tma-faint mt-2 text-center text-xs font-semibold">
        Lead Cat · v2.0 · {I18N[p.lang].appSub} 🐾
      </div>
    </div>
  )
}
