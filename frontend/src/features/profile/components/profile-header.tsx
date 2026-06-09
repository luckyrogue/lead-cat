import { useTmaAuth } from "@/shared/auth/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar } from "@/shared/ui/cat/primitives"

export function ProfileHeader() {
  const t = useTmaApp().t
  const { user } = useTmaAuth()

  return (
    <div className="mb-[22px] flex items-center gap-3.5 p-1">
      <Avatar name={user?.name ?? ""} size={62} />
      <div className="min-w-0 flex-1">
        <div className="font-display text-tma-text text-xl font-extrabold leading-[1.1]">
          {user?.name ?? ""}
        </div>
        <div className="text-tma-muted mb-1.5 truncate text-[13px]">
          {user?.email ?? ""}
        </div>
        {user?.role === "admin" && (
          <span className="font-display bg-tma-accent-soft text-tma-accent inline-flex items-center gap-[5px] rounded-full px-[9px] py-[3px] text-[11.5px] font-extrabold">
            👑 {t("role_admin")}
          </span>
        )}
      </div>
    </div>
  )
}
