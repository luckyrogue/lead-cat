import { useNavigate } from "@tanstack/react-router"
import { ColleagueSchedule } from "@/features/profile/components/colleague-schedule"
import { translate } from "@/shared/miniapp/i18n"
import { useMiniApp } from "@/shared/miniapp/context"
import { Overlay } from "@/components/miniapp-shell"

export function ColleagueSchedulePage() {
  const p = useMiniApp()
  const navigate = useNavigate()
  const goBack = () => void navigate({ to: "/" })

  return (
    <Overlay
      open
      onClose={goBack}
      onBack={goBack}
      title={translate(p.lang, "colleagueSchedule")}
    >
      <ColleagueSchedule />
    </Overlay>
  )
}
