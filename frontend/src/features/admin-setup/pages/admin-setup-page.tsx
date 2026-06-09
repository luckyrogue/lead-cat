import { useNavigate } from "@tanstack/react-router"
import { useMiniApp } from "@/shared/miniapp/context"
import { Overlay } from "@/components/miniapp-shell"
import { IntegrationsSection } from "../components/integrations-section"
import { ChatLinkSection } from "../components/chat-link-section"
import { MembersSection } from "../components/members-section"
import { AuditLogSection } from "../components/audit-log-section"

export function AdminSetupPage() {
  const { t } = useMiniApp()
  const navigate = useNavigate()
  const goBack = () => { void navigate({ to: "/profile/admin" }) }
  return (
    <Overlay open onClose={goBack} onBack={goBack} title={t("adminSetup" as never)}>
      <div className="flex flex-col gap-3 px-4 pb-7">
        <IntegrationsSection />
        <ChatLinkSection />
        <MembersSection />
        <AuditLogSection />
      </div>
    </Overlay>
  )
}
