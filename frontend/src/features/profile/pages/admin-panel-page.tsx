import { useNavigate } from "@tanstack/react-router"
import { AdminPanel } from "@/features/profile/components/admin-panel"
import { useTmaAuth } from "@/shared/auth/auth-context"
import { useMyMeetings } from "@/entities/meeting/queries"
import { RequireTmaRole } from "@/shared/auth/require-permission"
import { translate } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { Overlay } from "@/components/tma-shell"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { SettingsRow } from "@/features/profile/components/settings-row"
import { CatIcon } from "@/shared/ui/cat/primitives"

export function AdminPanelPage() {
  const p = useTmaApp()
  const { user } = useTmaAuth()
  const navigate = useNavigate()
  const { data: meetings = [] } = useMyMeetings("all")
  const goBack = () => void navigate({ to: "/profile" })

  return (
    <Overlay
      open
      onClose={goBack}
      onBack={goBack}
      title={translate(p.lang, "admin")}
    >
      <RequireTmaRole user={user} requirement="admin">
        <div className="px-4 pb-3">
          <SettingsGroup>
            <SettingsRow
              icon="settings"
              hue={210}
              label={translate(p.lang, "adminSetup" as never)}
              onClick={() => void navigate({ to: "/profile/admin/setup" })}
              last
              right={<span className="text-tma-faint"><CatIcon name="chevR" size={18} sw={2.2} /></span>}
            />
          </SettingsGroup>
        </div>
        <AdminPanel meetings={meetings} />
      </RequireTmaRole>
    </Overlay>
  )
}
