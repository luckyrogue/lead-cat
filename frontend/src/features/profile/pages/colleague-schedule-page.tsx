import { useNavigate } from "@tanstack/react-router"
import { ColleagueSchedule } from "@/features/profile/components/colleague-schedule"
import { translate } from "@/shared/tma/i18n"
import { useTmaApp } from "@/shared/tma/context"
import { Overlay } from "@/components/tma-shell"

export function ColleagueSchedulePage() {
  const p = useTmaApp()
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
