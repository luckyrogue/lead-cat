import { useNavigate } from "@tanstack/react-router"
import { useTmaApp } from "@/shared/tma/context"
import { Overlay } from "@/components/tma-shell"
import { IntegrationsSection } from "../components/integrations-section"
import { ChatLinkSection } from "../components/chat-link-section"
import { MembersSection } from "../components/members-section"
import { ScenariosSection } from "../components/scenarios-section"
import { AuditLogSection } from "../components/audit-log-section"

export function AdminSetupPage() {
  const { t } = useTmaApp()
  const navigate = useNavigate()
  const goBack = () => { void navigate({ to: "/profile/admin" }) }
  return (
    <Overlay open onClose={goBack} onBack={goBack} title={t("adminSetup" as never)}>
      <div className="flex flex-col gap-3 px-4 pb-7">
        <IntegrationsSection />
        <ChatLinkSection />
        <MembersSection />
        <ScenariosSection />
        <AuditLogSection />
      </div>
    </Overlay>
  )
}
