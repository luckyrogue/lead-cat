import { useNavigate } from "@tanstack/react-router"
import { AdminPanel } from "@/features/profile/components/admin-panel"
import { useTmaAuth } from "@/shared/auth/auth-context"
import { useMyMeetings } from "@/entities/meeting/queries"
import { RequireTmaRole } from "@/shared/auth/require-permission"
import { translate } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { Overlay } from "@/components/tma-shell"

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
        <AdminPanel meetings={meetings} />
      </RequireTmaRole>
    </Overlay>
  )
}
