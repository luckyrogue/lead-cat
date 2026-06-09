import { useMiniApp } from "@/shared/miniapp/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useMembers } from "@/entities/admin/queries"
import { useSyncMembers } from "@/entities/admin/mutations"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { cn } from "@/shared/lib/cn"

export function MembersSection() {
  const { t } = useMiniApp()
  const { data: members = [] } = useMembers()
  const syncMut = useSyncMembers()

  async function onSync() {
    try {
      const res = await syncMut.mutateAsync()
      toastSuccess(`${t("membersSynced" as never)}: +${res.added}`)
    } catch (err) {
      toastError(err, t("membersSyncFailed" as never))
    }
  }

  return (
    <SettingsGroup title={t("adminMembers" as never)}>
      <div className="flex flex-col gap-3 px-3 pb-3">
        <button
          type="button"
          disabled={syncMut.isPending}
          onClick={() => void onSync()}
          className="border-miniapp-accent bg-miniapp-accent self-start rounded-[12px] px-4 py-2 text-sm font-bold text-white disabled:cursor-not-allowed disabled:opacity-60"
        >
          {syncMut.isPending ? t("syncing" as never) : t("syncFromChatButton" as never)}
        </button>
        {members.length > 0 && (
          <div className="flex flex-col">
            {members.map((m, i) => (
              <div
                key={m.id}
                className={cn(
                  "flex items-center gap-2 py-2",
                  i < members.length - 1 && "border-miniapp-border border-b"
                )}
              >
                <div className="min-w-0 flex-1">
                  <div className="text-miniapp-text text-[14px] font-bold">{m.fullName}</div>
                  {m.telegramUsername && (
                    <div className="text-miniapp-muted truncate text-xs">@{m.telegramUsername}</div>
                  )}
                </div>
                <span className="text-miniapp-muted bg-miniapp-card-alt rounded-full px-2 py-0.5 text-[11px] font-bold">
                  {m.role}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </SettingsGroup>
  )
}
