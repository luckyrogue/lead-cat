import { useNavigate } from "@tanstack/react-router"
import { cn } from "@/shared/lib/cn"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { I18N } from "@/shared/miniapp/i18n"
import { useMiniApp } from "@/shared/miniapp/context"
import { canAccessMiniAppAdmin } from "@/shared/auth/module-policies"
import { useMiniAppAuth } from "@/shared/auth/auth-context"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { REMINDER_INTERVALS } from "@/entities/user-settings/constants"
import { useUserSettings } from "@/entities/user-settings/queries"
import { useUpdateReminderMinutes } from "@/entities/user-settings/mutations"
import { ProfileHeader } from "../components/profile-header"
import { SettingsGroup } from "../components/settings-group"
import { SettingsRow } from "../components/settings-row"

export function ProfilePage() {
  const p = useMiniApp()
  const { user } = useMiniAppAuth()
  const navigate = useNavigate()
  const t = p.t

  const { data: settings } = useUserSettings()
  const updateMut = useUpdateReminderMinutes()
  const current = settings?.reminderMinutes ?? []

  const toggleInterval = (minutes: number) => {
    const next = current.includes(minutes)
      ? current.filter((m) => m !== minutes)
      : [...current, minutes]
    updateMut.mutate(next, {
      onError: (err) => toastError(err, t("settingsSaveFailed")),
    })
  }

  return (
    <div className="px-4 pb-7">
      <ProfileHeader />

      <SettingsGroup title={t("reminders")}>
        <SettingsRow
          icon="bell"
          hue={45}
          label={t("reminders")}
          right={
            current.length > 0 ? (
              <span className="text-miniapp-muted text-xs font-bold">{current.length}</span>
            ) : undefined
          }
          last
        />
      </SettingsGroup>
      <div className="mx-1 -mt-3 mb-5 flex flex-col gap-2">
        {current.length === 0 && (
          <p className="text-miniapp-muted px-1 text-xs">{t("remindersOff")}</p>
        )}
        <div className="flex flex-wrap gap-2">
          {REMINDER_INTERVALS.map((it) => {
            const on = current.includes(it.value)
            return (
              <button
                key={it.value}
                type="button"
                onClick={() => toggleInterval(it.value)}
                className={cn(
                  "font-display cursor-pointer rounded-[11px] border-[1.5px] px-[13px] py-2 text-[13.5px] font-bold",
                  on
                    ? "border-miniapp-accent bg-miniapp-accent-soft text-miniapp-accent"
                    : "border-miniapp-border bg-miniapp-card text-miniapp-muted"
                )}
              >
                {on ? "✓ " : ""}
                {t(it.labelKey)}
              </button>
            )
          })}
        </div>
      </div>

      <SettingsGroup title="Предпочтения">
        <SettingsRow
          icon="globe"
          hue={180}
          label={t("language")}
          onClick={p.openLangPicker}
          right={
            <span className="text-miniapp-muted flex items-center gap-1.5 text-sm font-bold">
              {I18N[p.lang]._flag} {I18N[p.lang]._label}
              <span className="text-miniapp-faint">
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
            <span className="text-miniapp-muted flex items-center gap-1.5 text-sm font-bold">
              Алматы UTC+5
              <span className="text-miniapp-faint">
                <CatIcon name="chevR" size={16} sw={2.2} />
              </span>
            </span>
          }
          last
        />
      </SettingsGroup>

      {canAccessMiniAppAdmin(user) && (
        <SettingsGroup title={t("admin")}>
          <SettingsRow
            icon="shield"
            hue={25}
            label={t("admin")}
            onClick={() => void navigate({ to: "/profile/admin" })}
            right={
              <span className="text-miniapp-faint">
                <CatIcon name="chevR" size={18} sw={2.2} />
              </span>
            }
          />
          <SettingsRow
            icon="settings"
            hue={45}
            label="Настройка workspace"
            onClick={() => void navigate({ to: "/profile/admin/setup" })}
            right={
              <span className="text-miniapp-faint">
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
            <span className="text-miniapp-faint">
              <CatIcon name="chevR" size={18} sw={2.2} />
            </span>
          }
          last
        />
      </SettingsGroup>

      <div className="text-miniapp-faint mt-2 text-center text-xs font-semibold">
        Lead Cat · v2.0 · {I18N[p.lang].appSub} 🐾
      </div>
    </div>
  )
}
