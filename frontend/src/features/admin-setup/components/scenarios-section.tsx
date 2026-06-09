import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useScenarios } from "@/entities/admin/queries"
import { useToggleScenario, useRunScenario } from "@/entities/admin/mutations"
import { CatToggle } from "@/shared/ui/cat/primitives"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { cn } from "@/shared/lib/cn"

export function ScenariosSection() {
  const { t } = useTmaApp()
  const { data: scenarios = [] } = useScenarios()
  const toggleMut = useToggleScenario()
  const runMut = useRunScenario()

  async function onToggle(id: string, enabled: boolean) {
    try {
      await toggleMut.mutateAsync({ id, enabled })
    } catch (err) {
      toastError(err, t("scenarioToggleFailed" as never))
    }
  }

  async function onRun(id: string) {
    try {
      await runMut.mutateAsync(id)
      toastSuccess(t("scenarioRunStarted" as never))
    } catch (err) {
      toastError(err, t("scenarioRunFailed" as never))
    }
  }

  return (
    <SettingsGroup title={t("adminScenarios" as never)}>
      <div className="flex flex-col gap-3 px-3 pb-3">
        {scenarios.length === 0 && (
          <div className="text-tma-muted text-sm">{t("noScenarios" as never)}</div>
        )}
        {scenarios.map((s, i) => (
          <div
            key={s.id}
            className={cn(
              "flex items-center gap-3 py-2",
              i < scenarios.length - 1 && "border-tma-border border-b"
            )}
          >
            <div className="min-w-0 flex-1">
              <div className="text-tma-text text-[14px] font-bold">{s.name}</div>
              <div className="text-tma-muted text-xs">{s.schedule}</div>
            </div>
            <button
              type="button"
              disabled={runMut.isPending}
              onClick={() => void onRun(s.id)}
              className="border-tma-border text-tma-muted rounded-[8px] border px-2.5 py-1 text-xs font-bold disabled:opacity-60"
            >
              {t("runNow" as never)}
            </button>
            <CatToggle
              on={s.enabled}
              onChange={(enabled) => void onToggle(s.id, enabled)}
            />
          </div>
        ))}
      </div>
    </SettingsGroup>
  )
}
