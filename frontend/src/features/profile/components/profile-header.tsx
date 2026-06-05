import { useTmaAuth } from "@/features/auth/auth-context"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar } from "@/shared/ui/cat/primitives"

export function ProfileHeader() {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const t = p.t

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 14,
        marginBottom: 22,
        padding: "4px 4px",
      }}
    >
      <Avatar name={user?.name ?? ""} size={62} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontFamily: "var(--font-display)",
            fontWeight: 800,
            fontSize: 20,
            color: p.text,
            lineHeight: 1.1,
          }}
        >
          {user?.name ?? ""}
        </div>
        <div
          style={{
            fontSize: 13,
            color: p.muted,
            marginBottom: 6,
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          {user?.email ?? ""}
        </div>
        {user?.role === "admin" && (
          <span
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 5,
              fontSize: 11.5,
              fontWeight: 800,
              color: p.accent,
              background: p.accentSoft,
              padding: "3px 9px",
              borderRadius: 999,
              fontFamily: "var(--font-display)",
            }}
          >
            👑 {t("role_admin")}
          </span>
        )}
      </div>
    </div>
  )
}
