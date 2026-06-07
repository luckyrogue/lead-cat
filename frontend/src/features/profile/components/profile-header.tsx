import { useTmaAuth } from "@/features/auth/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar } from "@/shared/ui/cat/primitives"

export function ProfileHeader() {
  const t = useTmaApp().t
  const { user } = useTmaAuth()

  return (
    <div className="mb-[22px] flex items-center gap-3.5 p-1">
      <Avatar name={user?.name ?? ""} size={62} />
      <div className="min-w-0 flex-1">
        <div className="font-display text-xl font-extrabold leading-[1.1] text-tma-text">
          {user?.name ?? ""}
        </div>
        <div className="mb-1.5 truncate text-[13px] text-tma-muted">
          {user?.email ?? ""}
        </div>
        {user?.role === "admin" && (
          <span className="font-display inline-flex items-center gap-[5px] rounded-full bg-tma-accent-soft px-[9px] py-[3px] text-[11.5px] font-extrabold text-tma-accent">
            👑 {t("role_admin")}
          </span>
        )}
      </div>
    </div>
  )
}
